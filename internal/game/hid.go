package game

import "log/slog"

type HID struct {
	gr     *MemoryReader
	gi     *MemoryInjector
	logger *slog.Logger
}

func NewHID(gr *MemoryReader, gi *MemoryInjector, logger *slog.Logger) *HID {
	return &HID{
		gr:     gr,
		gi:     gi,
		logger: logger,
	}
}
