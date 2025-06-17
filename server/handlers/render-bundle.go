package handlers

import (
	"errors"
	"fmt"
	"github.com/lucas-gaitzsch/pdf-turtle/config"
	"github.com/lucas-gaitzsch/pdf-turtle/services"
	"mime/multipart"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/lucas-gaitzsch/pdf-turtle/services/bundles"
	"github.com/lucas-gaitzsch/pdf-turtle/services/pdf"
)

const (
	formDataKeyBundle         = "bundle"
	formDataKeyModel          = "model"
	formDataKeyTemplateEngine = "templateEngine"
	formDataKeyName           = "name"
	formDataKeyRenameFrom     = "renameFrom"
)

// RenderBundleHandler godoc
// @Summary      Render PDF from bundle including HTML(-Template) with model and assets provided in form-data (keys: bundle, model)
// @Description  Returns PDF file generated from bundle (Zip-File) of HTML or HTML template of body, header, footer and assets. The index.html file in the Zip-Bundle is required
// @Tags         Render HTML-Bundle
// @Accept       multipart/form-data
// @Produce      application/pdf
// @Param        bundle          formData  file    true   "Bundle Zip-File"
// @Param        model           formData  string  false  "JSON-Model for template (only required for template)"
// @Param        templateEngine  formData  string  false  "Template engine to use for template (only required for template)"
// @Success      200             "PDF File"
// @Router       /api/pdf/from/html-bundle/render [post]
func RenderBundleHandler(c *fiber.Ctx) error {
	ctx := c.UserContext()

	form, err := c.MultipartForm()
	if err != nil {
		return err
	}

	bundlesFromForm, ok := form.File[formDataKeyBundle]
	if !ok || len(bundlesFromForm) == 0 {
		return errors.New("no bundle data with key 'bundle' was attached in form data")
	}

	bundle, err := createBundle(bundlesFromForm)
	if err != nil {
		return err
	}

	err = bundle.TestIndexFile()
	if err != nil {
		return err
	}

	pdfService := pdf.NewPdfService(ctx)

	jsonModel, _ := getValueFromForm(form.Value, formDataKeyModel)
	templateEngine, _ := getValueFromForm(form.Value, formDataKeyTemplateEngine)

	pdfData, errRender := pdfService.PdfFromBundle(bundle, jsonModel, templateEngine)

	if errRender != nil {
		return errRender
	}

	return writePdf(c, "document.pdf", pdfData)
}

func createBundle(bundlesFromForm []*multipart.FileHeader) (*bundles.Bundle, error) {
	bundle := &bundles.Bundle{}

	for _, fb := range bundlesFromForm {

		if strings.HasPrefix(fb.Filename, "bundle") || fb.Header.Get("Content-Type") == "application/zip" || strings.HasSuffix(fb.Filename, ".zip") {
			reader, err := fb.Open()
			if err != nil {
				return nil, err
			}
			defer reader.Close()

			err = bundle.ReadFromZip(reader, fb.Size)

			if err != nil {
				return nil, err
			}
		} else {
			fp := &bundles.OpenerFileProxy{
				MultipartFileOpener: fb,
			}
			bundle.AddFile(fb.Filename, fp)
		}
	}
	return bundle, nil
}

// RenderBundleByNameHandler godoc
// @Summary      Render PDF from bundle by ID
// @Description  Returns PDF file generated from bundle (Zip-File) of HTML or HTML template of body, header, footer and assets.
// @Tags         Render HTML-Bundle
// @Accept       json
// @Produce      application/pdf
// @Param        id  path  string  true  "ID of the bundle"
// @Success      200  "PDF File"
// @Router       /api/pdf/from/html-bundle/{name} [post]
func RenderBundleByNameHandler(c *fiber.Ctx) error {
	ctx := c.UserContext()

	bundleProvider, ok := ctx.Value(config.ContextKeyBundleProviderService).(services.BundleProviderService)
	if !ok {
		return bundleProviderNotFound
	}
	name, err := url.PathUnescape(c.Params("name"))
	if err != nil {
		return fmt.Errorf("invalid name: %w", err)
	}

	info, err := bundleProvider.GetFromStore(ctx, name)
	if err != nil {
		return err
	}
	bundle := &bundles.Bundle{}
	if err = bundle.ReadFromZip(info.Data, info.Size); err != nil {
		return err
	}
	if err = bundle.TestIndexFile(); err != nil {
		return err
	}
	model := c.Body()
	pdfService := pdf.NewPdfService(ctx)
	pdfData, err := pdfService.PdfFromBundle(bundle, string(model), info.TemplateEngine)
	if err != nil {
		return err
	}

	return writePdf(c, info.Name+".pdf", pdfData)
}
