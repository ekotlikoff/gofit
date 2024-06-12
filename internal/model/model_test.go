package model

import (
	"fmt"
	"testing"

	"github.com/ekotlikoff/gofit/internal/static"
)

func TestAllMovementsHaveImages(t *testing.T) {
	loadMovementBank()
	for _, m := range movementBank {
		iterations := []string{"active"}
		if m.IterationNames != nil {
			iterations = m.IterationNames
		}
		iterations = append(iterations, "rest")
		for _, i := range iterations {
			imagePath := fmt.Sprintf("webpage/movement_images/%s/%s.png", m.Name, i)
			if _, err := static.WebpageStaticFS.Open(imagePath); err != nil {
				t.Errorf("%s is missing its image %s", m.Name, imagePath)
			}
		}
	}
}
