package observers

import (
	"fmt"
	"regexp"
	"time"

	flaggerv1 "github.com/weaveworks/flagger/pkg/apis/flagger/v1beta1"
	"github.com/weaveworks/flagger/pkg/metrics/providers"
)

var skipperQueries = map[string]string{
	"request-success-rate": `
	{{- $route := printf "kube_%s__%s.*__%s" namespace ingress service }}
	sum(
		rate(
			skipper_response_duration_seconds_bucket{
				namespace="{{ namespace }}",
				route=~"{{ $route }}",
				code!~"5..",
				le="+Inf"
			}[{{ interval }}]
		)
	)
	/ 
	sum(
		rate(
			skipper_response_duration_seconds_bucket{
				namespace="{{ namespace }}",
				route=~"{{ $route }}",
				le="+Inf"
			}[{{ interval }}]
		)
	)
	* 100`,
	"request-duration": `
	{{- $route := printf "kube_%s__%s.*__%s" namespace ingress service }}
	sum(
		rate(
			skipper_response_duration_seconds_sum{
				namespace="{{ namespace }}",
				route=~"{{ $route }}"
			}[{{ interval }}]
		)
	) 
	/ 
	sum(
		rate(
			skipper_response_duration_seconds_count{
				namespace="{{ namespace }}",
				route=~"{{ $route }}"
			}[{{ interval }}]
		)
	) 
	* 1000`,
}

// SkipperObserver Implentation for Skipper (https://github.com/zalando/skipper)
type SkipperObserver struct {
	client providers.Interface
}

// GetRequestSuccessRate return value for Skipper Request Success Rate
func (ob *SkipperObserver) GetRequestSuccessRate(model flaggerv1.MetricTemplateModel) (float64, error) {

	model = encodeModelForSkipper(model)

	query, err := RenderQuery(skipperQueries["request-success-rate"], model)
	if err != nil {
		return 0, fmt.Errorf("rendering query failed: %w", err)
	}

	value, err := ob.client.RunQuery(query)
	if err != nil {
		return 0, fmt.Errorf("running query failed: %w", err)
	}

	return value, nil
}

// GetRequestDuration return value for Skipper Request Duration
func (ob *SkipperObserver) GetRequestDuration(model flaggerv1.MetricTemplateModel) (time.Duration, error) {

	model = encodeModelForSkipper(model)

	query, err := RenderQuery(skipperQueries["request-duration"], model)
	if err != nil {
		return 0, fmt.Errorf("rendering query failed: %w", err)
	}

	value, err := ob.client.RunQuery(query)
	if err != nil {
		return 0, fmt.Errorf("running query failed: %w", err)
	}

	ms := time.Duration(int64(value)) * time.Millisecond
	return ms, nil
}

// encodeModelForSkipper replaces non word character in model with underscore to match route names
// https://github.com/zalando/skipper/blob/dd70bd65e7f99cfb5dd6b6f71885d9fe3b2707f6/dataclients/kubernetes/ingress.go#L101
func encodeModelForSkipper(model flaggerv1.MetricTemplateModel) flaggerv1.MetricTemplateModel {
	nonWord := regexp.MustCompile(`\W`)
	model.Ingress = nonWord.ReplaceAllString(model.Ingress, "_")
	model.Name = nonWord.ReplaceAllString(model.Name, "_")
	model.Namespace = nonWord.ReplaceAllString(model.Namespace, "_")
	model.Service = nonWord.ReplaceAllString(model.Service, "_")
	model.Target = nonWord.ReplaceAllString(model.Target, "_")
	return model
}