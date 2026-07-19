//go:build integration

package cms

import (
	"testing"

	domainCMS "github.com/freeDog-wy/go-backend-template/internal/domain/cms"
)

func TestLocaleRepositoryIntegrationDefaultLocale(t *testing.T) {
	fixture := newCMSIntegrationFixture(t)
	enLocale := &domainCMS.Locale{Code: "en-US", Name: "English", IsEnabled: true, SortOrder: 1}
	if err := fixture.repo.CreateLocale(fixture.ctx, enLocale); err != nil {
		t.Fatalf("create locale: %v", err)
	}
	if err := fixture.repo.SetDefaultLocale(fixture.ctx, enLocale.Code); err != nil {
		t.Fatalf("set default locale: %v", err)
	}
	if locale, err := fixture.repo.FindLocale(fixture.ctx, "zh-CN"); err != nil || locale.IsDefault {
		t.Fatalf("old default locale = %#v, %v", locale, err)
	}
	if locale, err := fixture.repo.FindLocale(fixture.ctx, "en-US"); err != nil || !locale.IsDefault {
		t.Fatalf("new default locale = %#v, %v", locale, err)
	}
}
