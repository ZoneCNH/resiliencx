package resiliencx

// ResilienceOption 配置函数，用于设置 ResilienceConfig 的选项。
type ResilienceOption func(*ResilienceConfig)

// ResilienceConfig 策略层全局配置。
type ResilienceConfig struct {
	Classifier Classifier
	Sink       Sink
}

// WithClassifier 设置错误分类器。
func WithClassifier(c Classifier) ResilienceOption {
	return func(cfg *ResilienceConfig) {
		cfg.Classifier = c
	}
}

// WithSink 设置事件接收器。
func WithSink(s Sink) ResilienceOption {
	return func(cfg *ResilienceConfig) {
		cfg.Sink = s
	}
}

// NewResilienceConfig 从选项构建策略层配置。
func NewResilienceConfig(opts ...ResilienceOption) ResilienceConfig {
	cfg := ResilienceConfig{
		Classifier: DefaultClassifier(),
		Sink:       NoopSink{},
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}
