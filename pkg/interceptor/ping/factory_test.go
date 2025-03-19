package ping

import (
	"context"
	"testing"
	"time"
)

func TestFactory(t *testing.T) {
	factory := CreateInterceptorFactory(WithInterval(10*time.Second), WithMaxHistory(100))
	if _, err := factory.NewInterceptor(context.Background(), "test"); err != nil {
		t.Errorf("failed to create interceptor with error: %v", err)
	}
}
