package schema

// MethodAttachmentReady is the notification method name for attachment download completion.
const MethodAttachmentReady = "attachment.ready"

// Attachment describes a file attached to a message.
// The content is not included — only metadata. Downloading is handled separately.
type Attachment struct {
	ContentType string `json:"content_type,omitempty"`
	FileName    string `json:"file_name,omitempty"`
	Size        uint32 `json:"size,omitempty"`
	Width       uint32 `json:"width,omitempty"`
	Height      uint32 `json:"height,omitempty"`
	Caption     string `json:"caption,omitempty"`
	LocalPath   string `json:"local_path,omitempty"`
}

// AttachmentReadyParams is the params payload for an "attachment.ready" notification.
// It is sent once per attachment after the file has been downloaded and stored locally.
// Account identifies which linked account received the message.
// Timestamp is the original message timestamp, uniquely identifying the message.
// Index is the zero-based position of the attachment within the message's attachment list.
// LocalPath is the absolute path to the decrypted file on disk.
type AttachmentReadyParams struct {
	Service   string `json:"service"`
	Account   string `json:"account"`
	Timestamp uint64 `json:"timestamp"`
	Index     int    `json:"index"`
	LocalPath string `json:"local_path"`
}
