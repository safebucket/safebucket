package notifier

type INotifier interface {
	NotifyFromTemplate(to string, subject string, templateName string, data interface{}) error
}
