from pdfminer.high_level import extract_text
import pdfplumber


class PDFTxtExtractor:
    def extract_simple_text(self, f_path: str):
        text = extract_text(f_path)
        return text

    def extract_data_with_tables(self, f_path: str):
        text = ''
        tables = ''
        with pdfplumber.open(f_path) as pdf:
            pages = pdf.pages
            for page in pages:  
                text += page.extract_text()
            
                tables = page.extract_tables()

        return text, tables
