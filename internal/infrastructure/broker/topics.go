package broker

var DefaultTopics = []string{
	"market.price.updated",
	"market.candle.created",
	"strategy.signal.generated",
	"risk.order.approved",
	"order.created",
	"execution.submitted",
	"portfolio.updated",
	"notification.send.requested",
}

var DefaultDLQTopic = "dead-letter"
