package v1

import "github.com/google/go-containerregistry/pkg/name"

func isValidImageName(imageName string) bool {
	_, err := name.ParseReference(imageName)
	return err == nil
}
