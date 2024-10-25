package jupyter

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func init() {
	lipgloss.SetColorProfile(termenv.ANSI256)
}

var (
	RedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#cc0000"))
	OrangeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff7c28"))
	YellowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#cc9500"))
	GreenStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#06cc00"))
	LightBlueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3cc5ff"))
	BlueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0c00cc"))
	LightPurpleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#d864ff"))
	PurpleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7400e0"))
	GrayStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#adadad"))

	NotificationStyles = []lipgloss.Style{RedStyle, OrangeStyle, GrayStyle, GreenStyle}
)

// Put together the requisite components of a 'response channel' key.
// This should NOT be called directly.
// Call either the 'getResponseChannelKeyForOriginalMessage' function or the 'getResponseChannelKeyForResponse' function.
//
// For the message type, we convert "{action}_request" message types to "{action}_reply"for use in the key.
func getResponseChannelKeyImpl(messageId string, messageType string, channel KernelSocketChannel) string {
	return fmt.Sprintf("%s-%s-%s", messageId, messageType, channel)
}

// Return the key that should be used to get/retrieve the 'response channel' for this message.
// This function accepts the original message that is/was sent TO the Jupyter components.
//
// For the message type, we convert "{action}_request" message types to "{action}_reply"for use in the key.
func getResponseChannelKeyFromRequest(originalMessage KernelMessage) string {
	var messageType = originalMessage.GetHeader().MessageType
	var channel = originalMessage.GetChannel()
	var messageId = originalMessage.GetHeader().MessageId

	if originalMessage.GetHeader().MessageType == CommCloseMessage {
		return ""
	}

	if !strings.HasSuffix(messageType.String(), "request") {
		panic(fmt.Sprintf("%s request %s has invalid message type: \"%s\"", channel, messageId, messageType))
	}

	// Since we're using the request to generate the key, we convert the message type to its reply variant.
	baseMessageType := messageType.getBaseMessageType()
	var messageTypeOfReply = baseMessageType + "reply"

	return getResponseChannelKeyImpl(messageId, messageTypeOfReply, channel)
}

// Return the key that should be used to get/retrieve the 'response channel' for this message.
// This function accepts a response that was sent BY Jupyter.
func getResponseChannelKeyFromReply(response KernelMessage) string {
	var messageType = response.GetHeader().MessageType
	var channel = response.GetChannel()
	var parentMessageId = response.GetParentHeader().MessageId

	if !strings.HasSuffix(messageType.String(), "reply") {
		panic(fmt.Sprintf("%s request %s has invalid message type: \"%s\"", channel, response.GetHeader().MessageId, messageType))
	}

	// Since we're using the reply to generate the key, we pass the message type as-is, without modification.
	return getResponseChannelKeyImpl(parentMessageId, messageType.String(), channel)
}
