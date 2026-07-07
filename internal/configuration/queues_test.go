package configuration

import (
	"strings"
	"testing"

	"github.com/safebucket/safebucket/internal/models"
)

func TestValidateUniqueQueueNames(t *testing.T) {
	tests := []struct {
		name    string
		queues  map[string]models.QueueConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "all distinct queue names",
			queues: map[string]models.QueueConfig{
				"notifications":   {Name: "safebucket-notifications"},
				"object_deletion": {Name: "safebucket-object-deletion"},
				"bucket_events":   {Name: "safebucket-bucket-events"},
			},
		},
		{
			name: "two queues share a name",
			queues: map[string]models.QueueConfig{
				"notifications":   {Name: "safebucket-shared"},
				"object_deletion": {Name: "safebucket-shared"},
				"bucket_events":   {Name: "safebucket-bucket-events"},
			},
			wantErr: true,
			errMsg:  "queue names must be unique",
		},
		{
			name: "all three queues share a name",
			queues: map[string]models.QueueConfig{
				"notifications":   {Name: "safebucket-shared"},
				"object_deletion": {Name: "safebucket-shared"},
				"bucket_events":   {Name: "safebucket-shared"},
			},
			wantErr: true,
			errMsg:  "queue names must be unique",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := models.EventsConfiguration{
				Type:   "jetstream",
				Queues: tt.queues,
			}
			err := validateUniqueQueueNames(events)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateUniqueQueueNames error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Fatalf("error message %q should contain %q", err.Error(), tt.errMsg)
			}
		})
	}
}
