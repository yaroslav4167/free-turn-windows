package main

import "testing"

func TestUpdateConfirmationAcceptsLinuxGTKResponse(t *testing.T) {
	for _, answer := range []string{"Yes", "Установить и перезапустить"} {
		confirmed, err := updateInstallationConfirmed(answer)
		if err != nil || !confirmed {
			t.Fatalf("affirmative answer %q ignored: %v", answer, err)
		}
	}
}

func TestUpdateConfirmationRejectsCancellation(t *testing.T) {
	for _, answer := range []string{"No", "Cancel", "Отмена", ""} {
		confirmed, err := updateInstallationConfirmed(answer)
		if err != nil || confirmed {
			t.Fatalf("cancellation %q accepted: %v", answer, err)
		}
	}
}

func TestUnknownUpdateConfirmationIsAnErrorNotSilentCancellation(t *testing.T) {
	confirmed, err := updateInstallationConfirmed("unexpected response")
	if err == nil || confirmed {
		t.Fatal("unknown response must be visible as an error")
	}
}
