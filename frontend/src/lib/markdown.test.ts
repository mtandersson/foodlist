import {describe, it, expect} from "vitest"
import {renderMarkdown} from "./markdown"

describe("renderMarkdown", () => {
  it("returns empty string for nullish or empty input", () => {
    expect(renderMarkdown("")).toBe("")
    expect(renderMarkdown(undefined)).toBe("")
    expect(renderMarkdown(null)).toBe("")
  })

  it("renders bold and italic", () => {
    expect(renderMarkdown("**bold** and *italic*")).toContain("<strong>bold</strong>")
    expect(renderMarkdown("**bold** and *italic*")).toContain("<em>italic</em>")
  })

  it("converts soft newlines to <br> via gfm breaks", () => {
    const out = renderMarkdown("line one\nline two")
    expect(out).toMatch(/line one\s*<br/)
  })

  it("renders bullet and ordered lists", () => {
    expect(renderMarkdown("- a\n- b")).toMatch(/<ul>[\s\S]*<li>a<\/li>[\s\S]*<li>b<\/li>[\s\S]*<\/ul>/)
    expect(renderMarkdown("1. a\n2. b")).toMatch(/<ol[\s\S]*<li>a<\/li>[\s\S]*<li>b<\/li>[\s\S]*<\/ol>/)
  })

  it("renders blockquotes", () => {
    expect(renderMarkdown("> note")).toContain("<blockquote>")
  })

  it("renders headings h3-h6 only (h1/h2 are reserved for title + sections)", () => {
    expect(renderMarkdown("### Heading")).toContain("<h3>Heading</h3>")
    // h1/h2 from markdown source are stripped by the allowlist but
    // their inner text is preserved (KEEP_CONTENT: true).
    const out = renderMarkdown("# Title\n## Sub")
    expect(out).not.toMatch(/<h1>/)
    expect(out).not.toMatch(/<h2>/)
    expect(out).toContain("Title")
    expect(out).toContain("Sub")
  })

  it("renders inline + fenced code", () => {
    expect(renderMarkdown("use `foo`")).toContain("<code>foo</code>")
    expect(renderMarkdown("```\nblock\n```")).toContain("<pre>")
  })

  describe("link sanitization", () => {
    it("allows http and https links and forces target+rel", () => {
      const out = renderMarkdown("[ex](https://example.com)")
      expect(out).toContain('href="https://example.com/"')
      expect(out).toContain('target="_blank"')
      expect(out).toContain('rel="noopener noreferrer"')
    })

    it("allows mailto links", () => {
      const out = renderMarkdown("[mail](mailto:hi@example.com)")
      expect(out).toContain("mailto:hi@example.com")
    })

    it("strips javascript: URLs", () => {
      const out = renderMarkdown("[x](javascript:alert(1))")
      expect(out).not.toContain("javascript:")
      expect(out).not.toMatch(/href=/)
    })

    it("strips data: URLs (XSS via data:text/html,...)", () => {
      const out = renderMarkdown("[x](data:text/html,<script>alert(1)</script>)")
      expect(out).not.toContain("data:")
      expect(out).not.toContain("<script")
    })

    it("strips vbscript: URLs", () => {
      const out = renderMarkdown("[x](vbscript:msgbox(1))")
      expect(out).not.toContain("vbscript:")
    })

    it("strips file: URLs", () => {
      const out = renderMarkdown("[x](file:///etc/passwd)")
      expect(out).not.toContain("file:")
    })

    it("strips relative URLs (description context has no base)", () => {
      const out = renderMarkdown("[x](/some/path)")
      expect(out).not.toContain('href="/some/path"')
    })

    it("strips protocol-relative URLs", () => {
      const out = renderMarkdown("[x](//evil.example.com)")
      expect(out).not.toContain("evil.example.com")
    })

    it("strips embedded credentials (user:pass@host phishing)", () => {
      const out = renderMarkdown("[x](https://user:pw@evil.example.com)")
      expect(out).not.toContain("user:")
      expect(out).not.toContain('href="')
    })

    it("strips control characters used to bypass scheme filters", () => {
      // The raw \t after "java" used to defeat regex-based filters
      // that match /^javascript:/.
      const out = renderMarkdown("[x](java\tscript:alert(1))")
      expect(out).not.toContain("javascript")
      expect(out).not.toMatch(/href=/)
    })
  })

  describe("tag allowlist (XSS surface)", () => {
    it("strips <script> tags entirely", () => {
      const out = renderMarkdown("hello <script>alert(1)</script> world")
      expect(out).not.toContain("<script")
      expect(out).not.toContain("alert(1)")
    })

    it("strips <img onerror=...> payloads", () => {
      const out = renderMarkdown('<img src=x onerror="alert(1)">')
      expect(out).not.toContain("<img")
      expect(out).not.toContain("onerror")
    })

    it("strips <iframe>", () => {
      const out = renderMarkdown('<iframe src="https://evil.example"></iframe>')
      expect(out).not.toContain("<iframe")
    })

    it("strips <style>", () => {
      const out = renderMarkdown("<style>body{background:red}</style>")
      expect(out).not.toContain("<style")
    })

    it("strips on* attributes from allowed tags", () => {
      const out = renderMarkdown('<a href="https://example.com" onclick="alert(1)">x</a>')
      expect(out).not.toContain("onclick")
    })
  })

  it("caps input length to defend against renderer DoS", () => {
    // 50k chars of asterisk-bold; should not crash and should produce
    // bounded output. Truncation happens before marked sees the input.
    const huge = "**a**".repeat(50_000)
    const out = renderMarkdown(huge)
    expect(out.length).toBeLessThan(50_000) // strict cap is 4096 + html overhead
  })
})
