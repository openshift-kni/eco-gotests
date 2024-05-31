package define

import (
	"html/template"
	"strings"

	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmmparams"
)

// MultiStageConfigMapContent returns the configmap multi-stage contents for a specified module name.
func MultiStageConfigMapContent(module string) map[string]string {
	data := map[string]interface{}{
		"Module": module,
	}

	templateInstance := template.Must(template.New("contents").Parse(kmmparams.MultistageContents))
	builder := &strings.Builder{}

	if err := templateInstance.Execute(builder, data); err != nil {
		panic(err)
	}

	content := builder.String()

	configmapContents := map[string]string{"dockerfile": content}

	return configmapContents
}
