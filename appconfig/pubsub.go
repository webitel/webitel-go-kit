package appconfig

// Pubsub holds message-broker configuration.
// Env vars: PUBSUB_URL, PUBSUB_DRIVER.
//
// Driver defaults to "rabbitmq" — PUBSUB_DRIVER only needs to be set to
// override. PUBSUB_URL is always required.
//
// Renamed from previous per-service convention:
//
//	pubsub.broker_url    (PUBSUB_BROKER_URL)    → pubsub.url    (PUBSUB_URL)
//	pubsub.broker_driver (PUBSUB_BROKER_DRIVER) → pubsub.driver (PUBSUB_DRIVER)
type Pubsub struct {
	URL    string `mapstructure:"url"`
	Driver string `mapstructure:"driver"`
}
