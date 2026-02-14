package events

// EventCollector is embedded in aggregates to collect domain events during state transitions.
type EventCollector struct {
	events []DomainEvent
}

// Record appends a domain event to the collector.
func (c *EventCollector) Record(event DomainEvent) {
	c.events = append(c.events, event)
}

// Events returns the collected domain events without clearing them.
func (c *EventCollector) Events() []DomainEvent {
	return c.events
}

// ClearEvents returns the collected domain events and clears the internal slice.
func (c *EventCollector) ClearEvents() []DomainEvent {
	collected := c.events
	c.events = nil
	return collected
}
