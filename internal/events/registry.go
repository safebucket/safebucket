package events

import "reflect"

// eventRegistry keeps a registry of all the events
// including their payload so they can be instantiated in the handler.
var eventRegistry = map[string]reflect.Type{
	BucketSharedWithName:        reflect.TypeOf(BucketSharedWith{}),
	BucketSharedWithPayloadName: reflect.TypeOf(BucketSharedWithPayload{}),
}
