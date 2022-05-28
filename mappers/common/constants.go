package common

import "time"

const (
	// RetryTimes for retry times
	RetryTimes = 5
	// RetryInterval for retry interval
	RetryInterval = 10 * time.Second
)

// Topics
const (
	TopicDevicePrefix = "$hw/events/device/"
	// TopicNodePrefix the topic prefix for membership event
	TopicNodePrefix = "$hw/events/node/"
)
