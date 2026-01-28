package main

import (
	"context"
	"crypto/tls"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/d0ugal/promexporter/app"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MosquittoCollector implements the app.Collector interface for MQTT metric collection
type MosquittoCollector struct {
	config     *MosquittoExporterConfig
	metrics    *MosquittoMetrics
	mqttClient mqtt.Client
	app        *app.App
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewMosquittoCollector creates a new Mosquitto collector
func NewMosquittoCollector(cfg *MosquittoExporterConfig, metrics *MosquittoMetrics, application *app.App) *MosquittoCollector {
	return &MosquittoCollector{
		config:  cfg,
		metrics: metrics,
		app:     application,
	}
}

// Start implements the Collector interface - starts MQTT connection and subscription
func (mc *MosquittoCollector) Start(ctx context.Context) {
	mc.ctx, mc.cancel = context.WithCancel(ctx)

	slog.Info("Starting Mosquitto collector",
		"broker", mc.config.Mosquitto.BrokerEndpoint,
		"client_id", mc.config.Mosquitto.ClientID,
		"tls_enabled", mc.config.Mosquitto.TLS.Enabled,
	)

	// Connect to broker in a goroutine
	go mc.connectToBroker()
}

// Stop implements the Collector interface - stops MQTT connection
func (mc *MosquittoCollector) Stop() {
	slog.Info("Stopping Mosquitto collector")

	if mc.cancel != nil {
		mc.cancel()
	}

	if mc.mqttClient != nil && mc.mqttClient.IsConnected() {
		mc.mqttClient.Disconnect(250)
		slog.Info("Disconnected from MQTT broker")
	}
}

// connectToBroker establishes connection to the MQTT broker with retry logic
func (mc *MosquittoCollector) connectToBroker() {
	opts := mqtt.NewClientOptions()
	opts.SetCleanSession(true)
	opts.AddBroker(mc.config.Mosquitto.BrokerEndpoint)

	// Set client ID if provided
	if mc.config.Mosquitto.ClientID != "" {
		opts.SetClientID(mc.config.Mosquitto.ClientID)
	}

	// Set username and password if provided
	if mc.config.Mosquitto.Username != "" {
		opts.SetUsername(mc.config.Mosquitto.Username)
		if !mc.config.Mosquitto.Password.IsEmpty() {
			opts.SetPassword(mc.config.Mosquitto.Password.Value())
		}
	}

	// Configure TLS if enabled
	if mc.config.Mosquitto.TLS.Enabled {
		if err := mc.configureTLS(opts); err != nil {
			slog.Error("Failed to configure TLS", "error", err)
			return
		}
	}

	// Set connection callbacks
	opts.OnConnect = mc.onConnect
	opts.OnConnectionLost = mc.onConnectionLost

	mc.mqttClient = mqtt.NewClient(opts)

	// Try to connect with retry logic
	for {
		select {
		case <-mc.ctx.Done():
			slog.Info("Connection attempt cancelled")
			return
		default:
			token := mc.mqttClient.Connect()
			if token.WaitTimeout(5 * time.Second) {
				if token.Error() == nil {
					slog.Info("Successfully connected to MQTT broker")
					return
				}
				slog.Error("Failed to connect to broker", "error", token.Error())
			} else {
				slog.Warn("Timeout connecting to broker", "endpoint", mc.config.Mosquitto.BrokerEndpoint)
			}
			time.Sleep(5 * time.Second)
		}
	}
}

// configureTLS sets up TLS configuration
func (mc *MosquittoCollector) configureTLS(opts *mqtt.ClientOptions) error {
	if mc.config.Mosquitto.TLS.CertFile == "" || mc.config.Mosquitto.TLS.KeyFile == "" {
		slog.Warn("TLS enabled but certificate or key file not provided")
		return nil
	}

	keyPair, err := tls.LoadX509KeyPair(mc.config.Mosquitto.TLS.CertFile, mc.config.Mosquitto.TLS.KeyFile)
	if err != nil {
		return err
	}

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{keyPair},
		InsecureSkipVerify: mc.config.Mosquitto.TLS.InsecureSkipVerify,
		ClientAuth:         tls.NoClientCert,
	}

	opts.SetTLSConfig(tlsConfig)

	// Warn if endpoint doesn't use TLS scheme
	endpoint := mc.config.Mosquitto.BrokerEndpoint
	if !strings.HasPrefix(endpoint, "ssl://") && !strings.HasPrefix(endpoint, "tls://") {
		slog.Warn("TLS configured but endpoint doesn't use ssl:// or tls:// scheme", "endpoint", endpoint)
	}

	return nil
}

// onConnect is called when successfully connected to the broker
func (mc *MosquittoCollector) onConnect(client mqtt.Client) {
	slog.Info("Connected to MQTT broker", "broker", mc.config.Mosquitto.BrokerEndpoint)

	// Subscribe to $SYS/# topic
	token := client.Subscribe("$SYS/#", 0, mc.messageHandler)
	if !token.WaitTimeout(10 * time.Second) {
		slog.Error("Timeout subscribing to topic $SYS/#")
		return
	}
	if err := token.Error(); err != nil {
		slog.Error("Failed to subscribe to topic $SYS/#", "error", err)
		return
	}

	slog.Info("Successfully subscribed to $SYS/# topic")
}

// onConnectionLost is called when connection to broker is lost
func (mc *MosquittoCollector) onConnectionLost(client mqtt.Client, err error) {
	slog.Error("Connection to MQTT broker lost", "error", err, "broker", mc.config.Mosquitto.BrokerEndpoint)

	// Reconnection will be handled automatically by the MQTT client library
	// or by our retry logic if needed
}

// messageHandler processes incoming MQTT messages
func (mc *MosquittoCollector) messageHandler(client mqtt.Client, msg mqtt.Message) {
	topic := msg.Topic()
	payload := string(msg.Payload())

	// Check if topic should be ignored
	if mc.metrics.ShouldIgnoreTopic(topic) {
		return
	}

	// Parse the metric name from topic
	metricName := parseTopic(topic)

	// Determine if this is a counter or gauge and process accordingly
	if mc.metrics.IsCounterTopic(topic) {
		mc.processCounterMetric(metricName, payload)
	} else {
		mc.processGaugeMetric(metricName, payload)
	}
}

// processCounterMetric processes a counter metric
func (mc *MosquittoCollector) processCounterMetric(metricName, payload string) {
	value := parseValue(payload)
	mc.metrics.SetCounterValue(metricName, value)
}

// processGaugeMetric processes a gauge metric
func (mc *MosquittoCollector) processGaugeMetric(metricName, payload string) {
	value := parseValue(payload)
	mc.metrics.SetGaugeValue(metricName, value)
}

// parseTopic converts an MQTT topic to a Prometheus metric name
func parseTopic(topic string) string {
	name := strings.Replace(topic, "$SYS/", "", 1)
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")
	return name
}

// parseValue extracts a numeric value from a payload string
func parseValue(payload string) float64 {
	var validValue = regexp.MustCompile(`-?\d{1,}[.]\d{1,}|\d{1,}`)
	// Get the first value in the string
	strArray := validValue.FindAllString(payload, 1)
	if len(strArray) > 0 {
		// Parse to float
		value, err := strconv.ParseFloat(strArray[0], 64)
		if err == nil {
			return value
		}
	}
	return 0
}
