import { serializeJsonLd } from "../seo";

/** Structured data for search engines; the content is built from our own constants only. */
export function JsonLd({ data }: { data: unknown }) {
  return (
    <script
      type="application/ld+json"
      dangerouslySetInnerHTML={{ __html: serializeJsonLd(data) }}
    />
  );
}
