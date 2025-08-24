package notifier

// INotifier defines the interface for sending notifications
type INotifier interface {
	NotifyFromTemplate(to string, subject string, templateName string, data interface{}) error
}
