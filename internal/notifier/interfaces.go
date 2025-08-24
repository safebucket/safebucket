package notifier

// INotifier defines the interface for sending notifications
type INotifier interface {
	// NotifyFromTemplate sends a notification using a template and data
	NotifyFromTemplate(to string, subject string, templateName string, data interface{}) error
}
