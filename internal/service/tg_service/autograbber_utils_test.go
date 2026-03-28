package tg_service

import (
	"myapp/internal/models"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func entityForSubstring(t *testing.T, text, substr, entityType, url string) models.MessageEntity {
	t.Helper()

	startByte := strings.Index(text, substr)
	if startByte < 0 {
		t.Fatalf("substring %q not found in text", substr)
	}
	endByte := startByte + len(substr)

	runeOffsets := byteOffsetsByRuneIndex(text)
	startRune, ok := runeIndexByByteOffset(runeOffsets, startByte)
	if !ok {
		t.Fatalf("failed to convert start byte offset %d to rune index", startByte)
	}
	endRune, ok := runeIndexByByteOffset(runeOffsets, endByte)
	if !ok {
		t.Fatalf("failed to convert end byte offset %d to rune index", endByte)
	}

	utf16Units := utf16UnitsByRuneIndex(text)

	return models.MessageEntity{
		Type:   entityType,
		Offset: utf16Units[startRune],
		Length: utf16Units[endRune] - utf16Units[startRune],
		Url:    url,
	}
}

func TestRebaseEntitiesToNewText_ExactMatch(t *testing.T) {
	srv := &TgService{l: zap.NewNop()}

	oldText := "Go here now\nContact @Pr11v_baitur"
	newText := "Go here now please\nContact @Pr11v_baitur"

	entities := []models.MessageEntity{
		entityForSubstring(t, oldText, "Go here now", "text_link", "https://example.org"),
		entityForSubstring(t, oldText, "@Pr11v_baitur", "mention", ""),
	}

	rebased := srv.rebaseEntitiesToNewText(entities, oldText, newText)
	if len(rebased) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(rebased))
	}

	gotLinkText, ok := extractEntityTextByUTF16(newText, rebased[0].Offset, rebased[0].Length)
	if !ok {
		t.Fatalf("failed to extract text_link from rebased entity")
	}
	if gotLinkText != "Go here now" {
		t.Fatalf("unexpected text_link text: got %q", gotLinkText)
	}

	gotMentionText, ok := extractEntityTextByUTF16(newText, rebased[1].Offset, rebased[1].Length)
	if !ok {
		t.Fatalf("failed to extract mention from rebased entity")
	}
	if gotMentionText != "@Pr11v_baitur" {
		t.Fatalf("unexpected mention text: got %q", gotMentionText)
	}
}

func TestRebaseEntitiesToNewText_TextLinkFallbackToFirstLine(t *testing.T) {
	srv := &TgService{l: zap.NewNop()}

	oldText := "REGISTER HERE\nContact @Pr11v_baitur"
	newText := "🚀 SIGN UP HERE 🚀\n\nContact @Pr11v_baitur"

	entities := []models.MessageEntity{
		entityForSubstring(t, oldText, "REGISTER HERE", "text_link", "https://example.org"),
		entityForSubstring(t, oldText, "@Pr11v_baitur", "mention", ""),
	}

	rebased := srv.rebaseEntitiesToNewText(entities, oldText, newText)
	if len(rebased) != 2 {
		t.Fatalf("expected 2 entities after fallback remap, got %d", len(rebased))
	}

	gotLinkText, ok := extractEntityTextByUTF16(newText, rebased[0].Offset, rebased[0].Length)
	if !ok {
		t.Fatalf("failed to extract fallback text_link from rebased entity")
	}
	if gotLinkText != "🚀 SIGN UP HERE 🚀" {
		t.Fatalf("fallback text_link should target first non-empty line, got %q", gotLinkText)
	}

	gotMentionText, ok := extractEntityTextByUTF16(newText, rebased[1].Offset, rebased[1].Length)
	if !ok {
		t.Fatalf("failed to extract mention from fallback rebased entity")
	}
	if gotMentionText != "@Pr11v_baitur" {
		t.Fatalf("unexpected mention text after fallback remap: got %q", gotMentionText)
	}
}

func TestRebaseEntitiesToNewText_MentionLichka_Exact(t *testing.T) {
	srv := &TgService{l: zap.NewNop()}

	oldText := "Связь со мной: @lichka"
	newText := "Связаться можно тут: @lichka"

	entities := []models.MessageEntity{
		entityForSubstring(t, oldText, "@lichka", "mention", ""),
	}

	rebased := srv.rebaseEntitiesToNewText(entities, oldText, newText)
	if len(rebased) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(rebased))
	}

	gotMentionText, ok := extractEntityTextByUTF16(newText, rebased[0].Offset, rebased[0].Length)
	if !ok {
		t.Fatalf("failed to extract mention from rebased entity")
	}
	if gotMentionText != "@lichka" {
		t.Fatalf("unexpected mention text: got %q", gotMentionText)
	}
}

func TestRebaseEntitiesToNewText_MentionLichka_MixedWithTextLinkFallback(t *testing.T) {
	srv := &TgService{l: zap.NewNop()}

	oldText := "РЕГИСТРАЦИЯ ЗДЕСЬ\nСвязь: @lichka"
	newText := "🚀 ОБНОВЛЕННАЯ ИНСТРУКЦИЯ 🚀\n\nСвязь: @lichka"

	entities := []models.MessageEntity{
		entityForSubstring(t, oldText, "РЕГИСТРАЦИЯ ЗДЕСЬ", "text_link", "https://example.org"),
		entityForSubstring(t, oldText, "@lichka", "mention", ""),
	}

	rebased := srv.rebaseEntitiesToNewText(entities, oldText, newText)
	if len(rebased) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(rebased))
	}

	gotLinkText, ok := extractEntityTextByUTF16(newText, rebased[0].Offset, rebased[0].Length)
	if !ok {
		t.Fatalf("failed to extract text_link from rebased entity")
	}
	if gotLinkText != "🚀 ОБНОВЛЕННАЯ ИНСТРУКЦИЯ 🚀" {
		t.Fatalf("unexpected text_link fallback target: got %q", gotLinkText)
	}

	gotMentionText, ok := extractEntityTextByUTF16(newText, rebased[1].Offset, rebased[1].Length)
	if !ok {
		t.Fatalf("failed to extract mention from rebased entity")
	}
	if gotMentionText != "@lichka" {
		t.Fatalf("unexpected mention text: got %q", gotMentionText)
	}
}