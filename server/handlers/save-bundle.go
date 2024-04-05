package handlers

import (
	"errors"
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/lucas-gaitzsch/pdf-turtle/config"
	"github.com/lucas-gaitzsch/pdf-turtle/services"
	"github.com/lucas-gaitzsch/pdf-turtle/services/bundles"

	"github.com/gofiber/fiber/v2"
)

// SaveHtmlBundleHandler godoc
// @Summary      Save HTML bundle to server
// @Description  Save HTML bundle to server, allowing to render PDFs from it at a later time
// @Tags         Save HTML-Bundle
// @Accept       multipart/form-data
// @Produce      application/json
// @Param        bundle          formData  file    true   "Bundle Zip-File"
// @Param        name            formData  string  true   "Name of the bundle"
// @Param        id              formData  string  false  "ID of the bundle"
// @Param        templateEngine  formData  string  false  "Template engine to use for template"
// @Success      200             "OK"
// @Router       /api/html-bundle/save [post]
func SaveHtmlBundleHandler(c *fiber.Ctx) error {
	ctx := c.UserContext()
	bundleProvider, ok := ctx.Value(config.ContextKeyBundleProviderService).(services.BundleProviderService)
	if !ok {
		return errors.New("bundle provider service not found in context")
	}

	form, err := c.MultipartForm()
	if err != nil {
		return err
	}

	bundlesFromForm, ok := form.File[formDataKeyBundle]
	if !ok || len(bundlesFromForm) == 0 {
		return errors.New("no bundle data with key 'bundle' was attached in form data")
	}

	fb := bundlesFromForm[0]

	f, err := fb.Open()
	if err != nil {
		return err
	}

	templateEngine, _ := getValueFromForm(form.Value, formDataKeyTemplateEngine)
	name, ok := getValueFromForm(form.Value, "name")
	bundleId, _ := getValueFromForm(form.Value, "id")
	if !ok {
		name = "template-" + randString()
	}

	info := bundles.BundleInfo{
		Id:             bundleId,
		Name:           name,
		TemplateEngine: templateEngine,
		FileName:       fb.Filename,
		Data:           f,
		Size:           fb.Size,
		ContentType:    fb.Header.Get("Content-Type"),
	}

	id, err := bundleProvider.Save(ctx, info)
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func randString() string {
	return strconv.FormatInt(rand.Int64(), 32) + "-" + time.Now().Truncate(time.Hour).String()
}
