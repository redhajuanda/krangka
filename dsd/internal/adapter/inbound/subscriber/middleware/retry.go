package middleware

import (
	"context"
	"errors"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
)

func Retry(maxRetries int, delay time.Duration, onRetry func(msg *message.Message, err error, attempt int, maxReached bool)) message.HandlerMiddleware {
	return func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			var lastErr error

			for i := 1; i <= maxRetries; i++ {
				msgs, err := h(msg)
				if err == nil {
					return msgs, nil
				}

				lastErr = err
				isMaxReached := i == maxRetries

				// Check if context is canceled and should not retry
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					// Call callback to mark as final attempt
					onRetry(msg, err, i, true)
					return nil, nil
				}

				// Check if this is a SkipRetryError - jika ya, langsung panggil onRetry dengan maxReached=true dan keluar
				if IsSkipRetryError(err) {
					// Unwrap error untuk mendapatkan error asli
					var skipErr *SkipRetryError
					if errors.As(err, &skipErr) {
						onRetry(msg, skipErr.Err, i, true)
					} else {
						onRetry(msg, err, i, true)
					}
					return nil, nil
				}

				// Call the retry callback with correct attempt number
				onRetry(msg, err, i, isMaxReached)

				// If max retries reached, return success (nil error) since message was acknowledged
				if isMaxReached {
					return nil, nil
				}

				// Check if context is canceled before sleeping
				if msg.Context().Err() != nil {
					return nil, nil
				}

				// Sleep before next retry
				time.Sleep(delay)
			}

			// This should never be reached, but just in case
			return nil, lastErr
		}
	}
}