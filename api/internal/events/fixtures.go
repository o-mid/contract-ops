package events

import "time"

func Fixtures() []Event {
	return []Event{
		{
			ID:            "evt_01HV1B00",
			Source:        "wirex-webhook",
			Type:          "card.transaction.settled",
			Status:        StatusProcessed,
			OccurredAt:    time.Date(2026, 7, 21, 8, 42, 0, 0, time.UTC),
			CorrelationID: "crl_card_8f3d",
		},
		{
			ID:            "evt_01HV1B01",
			Source:        "fireblocks-webhook",
			Type:          "transaction.completed",
			Status:        StatusProcessed,
			OccurredAt:    time.Date(2026, 7, 21, 8, 30, 0, 0, time.UTC),
			CorrelationID: "crl_tx_09a1",
		},
		{
			ID:            "evt_01HV1B02",
			Source:        "queue-data",
			Type:          "token.enrichment.requested",
			Status:        StatusPending,
			OccurredAt:    time.Date(2026, 7, 21, 8, 14, 0, 0, time.UTC),
			CorrelationID: "crl_token_29ab",
		},
		{
			ID:            "evt_01HV1B03",
			Source:        "onramp-webhook",
			Type:          "purchase.failed",
			Status:        StatusFailed,
			OccurredAt:    time.Date(2026, 7, 21, 7, 57, 0, 0, time.UTC),
			CorrelationID: "crl_onramp_62de",
		},
	}
}
