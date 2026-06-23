package llm

import (
	"context"
	"log/slog"
	"time"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/model"
	callbacksHelper "github.com/cloudwego/eino/utils/callbacks"
)

type timingCtxKey struct{}

// SetupGlobalCallbacks 注册 Eino 全局回调，监控所有 ChatModel 和 Embedding 调用。
// 在应用启动时调用一次即可。
func SetupGlobalCallbacks(logger *slog.Logger) {
	handler := callbacksHelper.NewHandlerHelper().
		ChatModel(&callbacksHelper.ModelCallbackHandler{
			OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *model.CallbackInput) context.Context {
				inputChars := 0
				for _, m := range input.Messages {
					inputChars += len(m.Content)
				}
				logger.Info("eino: chat start",
					"msg_count", len(input.Messages),
					"input_chars", inputChars,
					"has_tools", len(input.Tools) > 0,
				)
				return context.WithValue(ctx, timingCtxKey{}, time.Now())
			},
			OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *model.CallbackOutput) context.Context {
				elapsed := "N/A"
				if start, ok := ctx.Value(timingCtxKey{}).(time.Time); ok {
					elapsed = time.Since(start).String()
				}
				pc, cc := 0, 0
				if output.TokenUsage != nil {
					pc = output.TokenUsage.PromptTokens
					cc = output.TokenUsage.CompletionTokens
				}
				logger.Info("eino: chat done",
					"elapsed", elapsed,
					"output_chars", len(output.Message.Content),
					"prompt_tokens", pc,
					"completion_tokens", cc,
				)
				return ctx
			},
			OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
				elapsed := "N/A"
				if start, ok := ctx.Value(timingCtxKey{}).(time.Time); ok {
					elapsed = time.Since(start).String()
				}
				logger.Error("eino: chat error",
					"elapsed", elapsed,
					"error", err,
				)
				return ctx
			},
		}).
		Embedding(&callbacksHelper.EmbeddingCallbackHandler{
			OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *embedding.CallbackInput) context.Context {
				logger.Info("eino: embedding start",
					"text_count", len(input.Texts),
					"model", input.Config.Model,
				)
				return context.WithValue(ctx, timingCtxKey{}, time.Now())
			},
			OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *embedding.CallbackOutput) context.Context {
				elapsed := "N/A"
				if start, ok := ctx.Value(timingCtxKey{}).(time.Time); ok {
					elapsed = time.Since(start).String()
				}
				dims := 0
				if len(output.Embeddings) > 0 {
					dims = len(output.Embeddings[0])
				}
				pt := 0
				if output.TokenUsage != nil {
					pt = output.TokenUsage.PromptTokens
				}
				logger.Info("eino: embedding done",
					"elapsed", elapsed,
					"vector_count", len(output.Embeddings),
					"vector_dim", dims,
					"prompt_tokens", pt,
				)
				return ctx
			},
			OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
				elapsed := "N/A"
				if start, ok := ctx.Value(timingCtxKey{}).(time.Time); ok {
					elapsed = time.Since(start).String()
				}
				logger.Error("eino: embedding error",
					"elapsed", elapsed,
					"error", err,
				)
				return ctx
			},
		}).
		Handler()

	callbacks.AppendGlobalHandlers(handler)
}
