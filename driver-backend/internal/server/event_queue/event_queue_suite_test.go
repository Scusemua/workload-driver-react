package event_queue_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEventQueue(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "EventQueue Suite")
}
