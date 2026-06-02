export interface PrinterInfo {
  id: string;
  name: string;
  model: string;
  default_label_size?: string;
}

export function isNetworkPrinter(id: string): boolean {
  return id.startsWith("tcp://") || id.startsWith("socket://");
}

/** Ensure API / localStorage entries always include model for print requests. */
export function normalizePrinter(
  p: Partial<PrinterInfo> & Pick<PrinterInfo, "id" | "name">,
): PrinterInfo {
  return {
    id: p.id,
    name: p.name,
    model: p.model?.trim() || p.name || "Unknown",
    ...(p.default_label_size ? { default_label_size: p.default_label_size } : {}),
  };
}
