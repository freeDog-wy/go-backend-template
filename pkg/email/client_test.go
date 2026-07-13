package email

import "testing"

func TestNewRequiresExplicitMode(t *testing.T) {
	t.Parallel()

	if _, err := New(Config{}); err == nil {
		t.Fatal("New() expected an error for an omitted mode")
	}
}

func TestNewLogModeUsesDevSender(t *testing.T) {
	t.Parallel()

	sender, err := New(Config{Mode: ModeLog})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sender.(*DevSender); !ok {
		t.Fatalf("sender type = %T, want *DevSender", sender)
	}
}

func TestNewSMTPModeRequiresDeliveryFields(t *testing.T) {
	t.Parallel()

	if _, err := New(Config{Mode: ModeSMTP}); err == nil {
		t.Fatal("New() expected an error for incomplete smtp configuration")
	}
}
