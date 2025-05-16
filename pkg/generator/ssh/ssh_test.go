// Copyright External Secrets Inc. All Rights Reserved

package ssh

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func TestGenerate(t *testing.T) {
	type args struct {
		jsonSpec *apiextensions.JSON
		rsaGen   RSAGenerateFunc
	}
	tests := []struct {
		name    string
		g       *Generator
		args    args
		want    map[string][]byte
		wantErr bool
	}{
		{
			name: "no json spec should result in error",
			args: args{
				jsonSpec: nil,
			},
			wantErr: true,
		},
		{
			name: "invalid json spec should result in error",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`no json`),
				},
			},
			wantErr: true,
		},
		{
			name: "empty spec should return defaults",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`{}`),
				},
				rsaGen: func(bits int) (string, string, error) {
					assert.Equal(t, defaultBits, bits)
					return "foo", "bar", nil
				},
			},
			want: map[string][]byte{
				"id_rsa":     []byte(`foo`),
				"id_rsa.pub": []byte(`bar`),
			},
			wantErr: false,
		},
		{
			name: "spec should override defaults",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`{"spec":{"keyType":"RSA","rsaConfig":{"bits":2048}}}`),
				},
				rsaGen: func(bits int) (string, string, error) {
					assert.Equal(t, 2048, bits)
					return "foo", "bar", nil
				},
			},
			want: map[string][]byte{
				"id_rsa":     []byte(`foo`),
				"id_rsa.pub": []byte(`bar`),
			},
			wantErr: false,
		},
		{
			name: "invalid key type should result in error",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`{"spec":{"keyType":"FOO"}}`),
				},
				rsaGen: func(bits int) (string, string, error) {
					return "", "", nil
				},
			},
			wantErr: true,
		},
		{
			name: "generator error should be returned",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`{}`),
				},
				rsaGen: func(bits int) (string, string, error) {
					return "", "", errors.New("boom")
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{}
			got, _, err := g.generate(tt.args.jsonSpec, tt.args.rsaGen)
			if (err != nil) != tt.wantErr {
				t.Errorf("Generator.Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Generator.Generate() = %v, want %v", got, tt.want)
			}
		})
	}
}
