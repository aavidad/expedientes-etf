package catalogo

// MotivoCierre conserva las equivalencias de finalización de una versión.
type MotivoCierre struct {
	Codigo            string `json:"codigo"`
	Acta              string `json:"acta"`
	SPAF              string `json:"spaf"`
	AdmiteTexto       bool   `json:"admite_texto"`
	SinConsentimiento bool   `json:"sin_consentimiento"`
}

type CatalogoCierre struct {
	VersionID string         `json:"version_id"`
	Motivos   []MotivoCierre `json:"motivos"`
}
