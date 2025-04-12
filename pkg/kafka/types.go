package kafka

type Broker struct {
	ID   int32  `json:"id"`
	Addr string `json:"addr"`
	Rack string `json:"rack"`
}

type PartitionOffset struct {
	Partition   int32 `json:"partition"`
	StartOffset int64 `json:"startOffset"`
	EndOffset   int64 `json:"endOffset"`
}

type GroupPartitionInfo struct {
	Topic         string `json:"topic,omitempty"`
	Partition     int32  `json:"partition"`
	CurrentOffset int64  `json:"currentOffset"`
	LogEndOffset  int64  `json:"logEndOffset"`
	Lag           int64  `json:"lag,omitempty"`
}

type GroupInfo struct {
	GroupID string               `json:"groupId"`
	Offsets []GroupPartitionInfo `json:"offsets"`
}

type MessagePartitionOffset struct {
	Partition int
	Offset    int
}
