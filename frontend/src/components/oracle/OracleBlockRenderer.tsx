import type { OracleBlock } from "../../api/types";
import { HeaderBlock } from "./blocks/HeaderBlock";
import { SymbolListBlock } from "./blocks/SymbolListBlock";
import { GraphBlock } from "./blocks/GraphBlock";
import { TableBlock } from "./blocks/TableBlock";
import { TextBlock } from "./blocks/TextBlock";
import { TruncationBlock } from "./blocks/TruncationBlock";

interface Props {
  block: OracleBlock;
}

export function OracleBlockRenderer({ block }: Props) {
  switch (block.type) {
    case "header":
      return <HeaderBlock data={block.data} />;
    case "symbol_list":
      return <SymbolListBlock data={block.data} />;
    case "graph":
      return <GraphBlock data={block.data} />;
    case "table":
      return <TableBlock data={block.data} />;
    case "text":
      return <TextBlock data={block.data} />;
    case "truncation":
      return <TruncationBlock data={block.data} />;
    default:
      return null;
  }
}
