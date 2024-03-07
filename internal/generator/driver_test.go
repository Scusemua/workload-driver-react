package generator

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestWaitGroup(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Driver")
}
