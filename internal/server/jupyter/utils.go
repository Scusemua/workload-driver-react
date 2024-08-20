package jupyter

import (
	"fmt"

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
func getResponseChannelKeyImpl(messageId string, messageType MessageType, channel KernelSocketChannel) string {
	return fmt.Sprintf("%s-%s-%s", messageId, messageType, channel)
}

// Return the key that should be used to get/retrieve the 'response channel' for this message.
// This function accepts the original message that is/was sent TO the Jupyter components.
func getResponseChannelKeyFromRequest(originalMessage KernelMessage) string {
	var messageType MessageType = originalMessage.GetHeader().MessageType
	var channel KernelSocketChannel = originalMessage.GetChannel()
	var messageId = originalMessage.GetHeader().MessageId

	return getResponseChannelKeyImpl(messageId, messageType, channel)
}

// Return the key that should be used to get/retrieve the 'response channel' for this message.
// This function accepts a response that was sent BY Jupyter.
func getResponseChannelKeyFromReply(response KernelMessage) string {
	var messageType MessageType = response.GetHeader().MessageType
	var channel KernelSocketChannel = response.GetChannel()
	var parentMessageId = response.GetParentHeader().MessageId

	return getResponseChannelKeyImpl(parentMessageId, messageType, channel)
}
