package handler

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/redhajuanda/komon/logger"
	"github.com/redhajuanda/krangka/configs"
)

// NoteHandler is a handler for the note events
type NoteHandler struct {
	cfg        *configs.Config
	log        logger.Logger
	subscriber message.Subscriber
}

// NewNoteHandler creates a new NoteHandler
func NewNoteHandler(cfg *configs.Config, log logger.Logger, subscriber message.Subscriber) *NoteHandler {
	return &NoteHandler{
		cfg:        cfg,
		log:        log,
		subscriber: subscriber,
	}
}

// RegisterRoutes registers the routes for the NoteHandler
func (h *NoteHandler) RegisterRoutes(router *message.Router) {

	router.AddConsumerHandler("NOTE_CREATED", "note.created", h.subscriber, h.HandleNoteCreated)
	router.AddConsumerHandler("NOTE_UPDATED", "note.updated", h.subscriber, h.HandleNoteUpdated)
	router.AddConsumerHandler("NOTE_DELETED", "note.deleted", h.subscriber, h.HandleNoteDeleted)

}

// HandleNoteCreated handles the note created event
func (h *NoteHandler) HandleNoteCreated(msg *message.Message) error {

	fmt.Println("uuid: ", msg.UUID)
	fmt.Println("metadata: ", msg.Metadata)
	fmt.Println("payload: ", string(msg.Payload))

	return nil
}

// HandleNoteUpdated handles the note updated event
func (h *NoteHandler) HandleNoteUpdated(msg *message.Message) error {

	fmt.Println("uuid: ", msg.UUID)
	fmt.Println("metadata: ", msg.Metadata)
	fmt.Println("updated note: ", string(msg.Payload))
	return nil

}

// HandleNoteDeleted handles the note deleted event
func (h *NoteHandler) HandleNoteDeleted(msg *message.Message) error {

	fmt.Println("uuid: ", msg.UUID)
	fmt.Println("metadata: ", msg.Metadata)
	fmt.Println("deleted note: ", string(msg.Payload))
	return nil

}
