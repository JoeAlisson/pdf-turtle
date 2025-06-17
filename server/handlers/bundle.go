package handlers

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"net/url"

	"github.com/gofiber/fiber/v2"
	"github.com/lucas-gaitzsch/pdf-turtle/config"
	"github.com/lucas-gaitzsch/pdf-turtle/services"
	"github.com/lucas-gaitzsch/pdf-turtle/services/bundles"
)

var bundleProviderNotFound = errors.New("bundle provider service not found in context")

// SaveHtmlBundleHandler godoc
// @Summary      Save HTML bundle to server
// @Description  Save HTML bundle to server, allowing to render PDFs from it at a later time
// @Tags         Save HTML-Bundle
// @Accept       multipart/form-data
// @Produce      application/json
// @Param        bundle          formData  file     true   "Bundle Zip-File"
// @Param        name            formData  string   true   "Name of the bundle"
// @Param        templateEngine  formData  string   true   "Template engine to use for template"
// @Param        rename  	     formData  boolean  false  "If true, the bundle will be renamed"
// @Success      201             "Created"
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
	renameFrom, _ := getValueFromForm(form.Value, formDataKeyRenameFrom)
	name, ok := getValueFromForm(form.Value, formDataKeyName)
	if !ok {
		name = "template-" + randString()
	}

	info := bundles.Info{
		Name:           name,
		TemplateEngine: templateEngine,
		Data:           f,
		Size:           fb.Size,
		ContentType:    fb.Header.Get("Content-Type"),
		RenameFrom:     renameFrom,
	}

	err = bundleProvider.Save(ctx, info)
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"name": name})
}

// GetHtmlBundleHandler godoc
// @Summary      Get HTML bundle from server
// @Description  Get HTML bundle from server, allowing to render PDFs from it at a later time
// @Tags         Get HTML-Bundle
// @Produce      multipart/form-data
// @Param        id  path  string  true  "ID of the bundle"
// @Success      200  "OK"
// @Router       /api/html-bundle/{name} [get]
func GetHtmlBundleHandler(c *fiber.Ctx) error {
	ctx := c.UserContext()
	bundleProvider, ok := ctx.Value(config.ContextKeyBundleProviderService).(services.BundleProviderService)
	if !ok {
		return bundleProviderNotFound
	}

	name, err := url.PathUnescape(c.Params("name"))
	if err != nil {
		return err
	}

	info, err := bundleProvider.GetFromStore(ctx, name)
	if err != nil {
		return err
	}

	w := multipart.NewWriter(c)

	c.Set(fiber.HeaderContentType, w.FormDataContentType())

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
		escapeQuotes(formDataKeyBundle), "bundle"))
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

	if err = w.WriteField(formDataKeyTemplateEngine, info.TemplateEngine); err != nil {
		return err
	}
	return w.Close()
}

// ListHtmlBundlesInfoHandler godoc
// @Summary      List HTML bundles from server
// @Description  List HTML bundles from server, allowing to render PDFs from it at a later time
// @Tags         List HTML-Bundles Info
// @Query        prefix  string  false  "Prefix to filter bundles by name"
// @Produce      application/json
// @Success      200  "OK"
// @Router       /api/html-bundle [get]
func ListHtmlBundlesInfoHandler(c *fiber.Ctx) error {
	ctx := c.UserContext()
	bundleProvider, ok := ctx.Value(config.ContextKeyBundleProviderService).(services.BundleProviderService)
	if !ok {
		return bundleProviderNotFound
	}

	prefix := c.Query("prefix", "")

	list, err := bundleProvider.ListInfoFromStore(ctx, prefix)
	if err != nil {
		return err
	}
	return c.JSON(list)

}
