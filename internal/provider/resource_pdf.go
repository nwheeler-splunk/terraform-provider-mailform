package provider

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"io/ioutil"
	"os"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/jung-kurt/gofpdf"
)

func resourcePDF() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Render a PDF and write to a local file.",

		CreateContext: resourcePDFCreate,
		ReadContext:   resourcePDFRead,
		DeleteContext: resourcePDFDelete,

		Schema: map[string]*schema.Schema{
			"header": {
				Description: "Header/title of PDF",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"content": {
				Description: "Content of PDF",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"filename": {
				Description: "The path to the PDF file that will be created",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
		},
	}
}

func resourcePDFCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics

	pdf := gofpdf.New(gofpdf.OrientationPortrait, "mm", gofpdf.PageSizeLetter, "")

	header := d.Get("header").(string)
	content := d.Get("content").(string)
	filename := d.Get("filename").(string)

	pdf.AddPage()
	pdf.SetTitle(d.Get("header").(string), false)
	pdf.SetFont("Arial", "B", 16)
	// Calculate width of title and position
	wd := pdf.GetStringWidth(header) + 6
	pdf.SetX((210 - wd) / 2)
	// Title
	pdf.CellFormat(wd, 9, header, "", 1, "C", false, 0, "")
	// Line break
	pdf.Ln(10)
	pdf.SetFont("Arial", "", 11)
	pdf.SetAutoPageBreak(true, 2.00)
	// Write ze content
	pdf.Write(8, content)

	err := pdf.OutputFileAndClose(filename)
	if err != nil {
		return diag.FromErr(err)
	}

	outputContent, err := ioutil.ReadFile(filename)
	if err != nil {
		return diag.FromErr(err)
	}

	checksum := sha1.Sum([]byte(outputContent))
	d.SetId(hex.EncodeToString(checksum[:]))

	tflog.Trace(ctx, "created a pdf resource")

	return diags
}

func resourcePDFRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	// If the output file doesn't exist, mark the resource for creation.
	outputPath := d.Get("filename").(string)
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		d.SetId("")
		return nil
	}

	// Verify that the content of the destination file matches the content we
	// expect. Otherwise, the file might have been modified externally, and we
	// must reconcile.
	outputContent, err := ioutil.ReadFile(outputPath)
	if err != nil {
		return diag.FromErr(err)
	}

	outputChecksum := sha1.Sum(outputContent)
	if hex.EncodeToString(outputChecksum[:]) != d.Id() {
		d.SetId("")
		return nil
	}

	return nil
}

func resourcePDFDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	os.Remove(d.Get("filename").(string))
	return nil
}
