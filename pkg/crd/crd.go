package crd

import (
	"encoding/json"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	v1 "github.com/crossplane/crossplane/apis/apiextensions/v1"
	"github.com/giantswarm/crd-docs-generator/internal/xcrd"
	"github.com/giantswarm/microerror"
	"github.com/pkg/errors"
	"io/ioutil"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"strings"
)

type Generator struct {
	Scheme *runtime.Scheme
}

func NewGenerator() *Generator {
	sch := runtime.NewScheme()
	_ = v1.AddToScheme(sch)
	_ = apiextensionsv1.AddToScheme(sch)
	return &Generator{
		Scheme: sch,
	}
}

// Read reads a CRD YAML file and returns the Custom Resource Definition objects it represents.
func (g *Generator) Read(filePath string) (resourceDefinitions []runtime.Object, error error) {
	decode := serializer.NewCodecFactory(g.Scheme).UniversalDeserializer().Decode

	yamlBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, microerror.Maskf(CouldNotReadCRDFileError, err.Error())
	}

	// Split by "---"
	parts := strings.Split(string(yamlBytes), "\n---\n")
	for _, crdYAMLString := range parts {
		crdYAMLBytes := []byte(crdYAMLString)
		obj, _, err := decode(crdYAMLBytes, nil, nil)
		if err != nil {
			return nil, microerror.Maskf(CouldNotParseCRDFileError, err.Error())
		}

		resourceDefinitions = append(resourceDefinitions, obj)
	}

	return resourceDefinitions, nil
}

const (
	CategoryComposite = "composite"
)

const (
	errFmtGetProps     = "cannot get %q properties from validation schema"
	errParseValidation = "cannot parse validation schema"
)

// ForCompositeResource derives the CustomResourceDefinition for a composite
// resource from the supplied CompositeResourceDefinition.
func ForCompositeResource(rd *v1.CompositeResourceDefinition) (*apiextensionsv1.CustomResourceDefinition, error) {
	crd := &apiextensionsv1.CustomResourceDefinition{
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Scope:    apiextensionsv1.ClusterScoped,
			Group:    rd.Spec.Group,
			Names:    rd.Spec.Names,
			Versions: make([]apiextensionsv1.CustomResourceDefinitionVersion, len(rd.Spec.Versions)),
		},
	}

	crd.SetName(rd.GetName())
	crd.SetLabels(rd.GetLabels())
	crd.SetAnnotations(rd.GetAnnotations())
	crd.SetOwnerReferences([]metav1.OwnerReference{meta.AsController(
		meta.TypedReferenceTo(rd, v1.CompositeResourceDefinitionGroupVersionKind),
	)})

	crd.Spec.Names.Categories = append(crd.Spec.Names.Categories, CategoryComposite)

	for i, vr := range rd.Spec.Versions {
		crd.Spec.Versions[i] = apiextensionsv1.CustomResourceDefinitionVersion{
			Name:                     vr.Name,
			Served:                   vr.Served,
			Storage:                  vr.Referenceable,
			AdditionalPrinterColumns: append(vr.AdditionalPrinterColumns, xcrd.CompositeResourcePrinterColumns()...),
			Schema: &apiextensionsv1.CustomResourceValidation{
				OpenAPIV3Schema: xcrd.BaseProps(),
			},
			Subresources: &apiextensionsv1.CustomResourceSubresources{
				Status: &apiextensionsv1.CustomResourceSubresourceStatus{},
			},
		}

		p, required, err := getProps("spec", vr.Schema)
		if err != nil {
			return nil, errors.Wrapf(err, errFmtGetProps, "spec")
		}
		specProps := crd.Spec.Versions[i].Schema.OpenAPIV3Schema.Properties["spec"]
		specProps.Required = append(specProps.Required, required...)
		for k, v := range p {
			specProps.Properties[k] = v
		}
		for k, v := range xcrd.CompositeResourceSpecProps() {
			specProps.Properties[k] = v
		}
		crd.Spec.Versions[i].Schema.OpenAPIV3Schema.Properties["spec"] = specProps

		statusP, statusRequired, err := getProps("status", vr.Schema)
		if err != nil {
			return nil, errors.Wrapf(err, errFmtGetProps, "status")
		}
		statusProps := crd.Spec.Versions[i].Schema.OpenAPIV3Schema.Properties["status"]
		statusProps.Required = statusRequired
		for k, v := range statusP {
			statusProps.Properties[k] = v
		}
		for k, v := range xcrd.CompositeResourceStatusProps() {
			statusProps.Properties[k] = v
		}
		crd.Spec.Versions[i].Schema.OpenAPIV3Schema.Properties["status"] = statusProps
	}

	return crd, nil
}

func getProps(field string, v *v1.CompositeResourceValidation) (map[string]apiextensionsv1.JSONSchemaProps, []string, error) {
	if v == nil {
		return nil, nil, nil
	}

	s := &apiextensionsv1.JSONSchemaProps{}
	if err := json.Unmarshal(v.OpenAPIV3Schema.Raw, s); err != nil {
		return nil, nil, errors.Wrap(err, errParseValidation)
	}

	spec, ok := s.Properties[field]
	if !ok {
		return nil, nil, nil
	}

	return spec.Properties, spec.Required, nil
}
