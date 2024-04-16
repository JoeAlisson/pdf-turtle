package handlers

import (
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/lucas-gaitzsch/pdf-turtle/config"
	"github.com/lucas-gaitzsch/pdf-turtle/services"
	"github.com/lucas-gaitzsch/pdf-turtle/services/bundles"
	"io"
	"mime/multipart"
	"net/textproto"
)

var bundleProviderNotFound = errors.New("bundle provider service not found in context")

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
// @Router       /api/html-bundle [post]
func SaveHtmlBundleHandler(c *fiber.Ctx) error {
	ctx := c.UserContext()
	bundleProvider, ok := ctx.Value(config.ContextKeyBundleProviderService).(services.BundleProviderService)
	if !ok {
		return bundleProviderNotFound
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
	name, ok := getValueFromForm(form.Value, formDataKeyName)
	bundleId, _ := getValueFromForm(form.Value, formDataKeyId)
	if !ok {
		name = "template-" + randString()
	}

	info := bundles.Info{
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

// GetHtmlBundleHandler godoc
// @Summary      Get HTML bundle from server
// @Description  Get HTML bundle from server, allowing to render PDFs from it at a later time
// @Tags         Get HTML-Bundle
// @Produce      multipart/form-data
// @Param        id  path  string  true  "ID of the bundle"
// @Success      200  "OK"
// @Router       /api/html-bundle/{id} [get]
func GetHtmlBundleHandler(c *fiber.Ctx) error {
	ctx := c.UserContext()
	bundleProvider, ok := ctx.Value(config.ContextKeyBundleProviderService).(services.BundleProviderService)
	if !ok {
		return bundleProviderNotFound
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return err
	}

	info, err := bundleProvider.GetFromStore(ctx, id)
	if err != nil {
		return err
	}

	w := multipart.NewWriter(c)
	defer w.Close()

	c.Set(fiber.HeaderContentType, w.FormDataContentType())

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
		escapeQuotes(formDataKeyBundle), escapeQuotes(info.FileName)))
	h.Set("Content-Type", info.ContentType)

	ff, err := w.CreatePart(h)
	if err != nil {
		return err
	}

	if _, err = io.Copy(ff, info.Data); err != nil {
		return err
	}

	if err = w.WriteField(formDataKeyName, info.Name); err != nil {
		return err
	}

	if err = w.WriteField(formDataKeyId, info.Id); err != nil {
		return err
	}

	if err = w.WriteField(formDataKeyTemplateEngine, info.TemplateEngine); err != nil {
		return err
	}
	return nil
}

// ListHtmlBundlesInfoHandler godoc
// @Summary      List HTML bundles from server
// @Description  List HTML bundles from server, allowing to render PDFs from it at a later time
// @Tags         List HTML-Bundles Info
// @Produce      application/json
// @Success      200  "OK"
// @Router       /api/html-bundle [get]
func ListHtmlBundlesInfoHandler(c *fiber.Ctx) error {
	ctx := c.UserContext()
	bundleProvider, ok := ctx.Value(config.ContextKeyBundleProviderService).(services.BundleProviderService)
	if !ok {
		return bundleProviderNotFound
	}
	list, err := bundleProvider.ListInfoFromStore(ctx)
	if err != nil {
		return err
	}
	return c.JSON(list)

}
