package configuration

import (
	"fmt"
	"sort"

	"github.com/safebucket/safebucket/internal/models"
)

func validateUniqueQueueNames(events models.EventsConfiguration) error {
	keys := make([]string, 0, len(events.Queues))
	for key := range events.Queues {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	seen := make(map[string]string, len(keys))
	for _, key := range keys {
		name := events.Queues[key].Name
		if existingKey, ok := seen[name]; ok {
			return fmt.Errorf(
				"events.queues %q and %q both use the queue name %q: queue names must be unique",
				existingKey, key, name,
			)
		}
		seen[name] = key
	}
	return nil
}
