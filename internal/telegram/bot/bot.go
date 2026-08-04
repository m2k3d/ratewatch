package bot

import (
	"fmt"
	"ratewatch/internal/telegram"
)

func Logic(result []telegram.Update, token string, offset int) (int, error) {
	for _, u := range result {
		err := telegram.SendMessage(u.Message.Chat.ID, u.Message.Text, token)
		if err != nil {
			offset = u.UpdateID + 1
			return offset, fmt.Errorf("can't send message: %w", err)
		}

		offset = u.UpdateID + 1
	}

	return offset, nil
}
