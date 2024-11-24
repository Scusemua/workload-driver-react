package event_queue_test

import (
	"github.com/google/uuid"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"github.com/scusemua/workload-driver-react/m/v2/internal/mock_domain"
	"go.uber.org/mock/gomock"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var mockCtrl *gomock.Controller

func createEvent(name domain.EventName, sessionId string, index uint64, timestamp time.Time, mockCtrl *gomock.Controller) *domain.Event {
	data := mock_domain.NewMockSessionMetadata(mockCtrl)
	data.EXPECT().GetPod().AnyTimes().Return(sessionId)

	return &domain.Event{
		Name:        name,
		GlobalIndex: index,
		LocalIndex:  int(index),
		ID:          uuid.NewString(),
		Timestamp:   timestamp,
		SessionId:   sessionId,
		Data:        data,
	}
}

func TestEventQueue(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "EventQueue Suite")
}
