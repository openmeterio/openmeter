import type { ModelProperty, Type } from "@typespec/compiler";
import { useTsp } from "@typespec/emitter-framework";
import { callPart, isBuiltIn } from "./utils.jsx";

export function zodDescriptionParts(type: Type, member?: ModelProperty) {
  const { $ } = useTsp();

  const sources: (Type | ModelProperty)[] = [];
  if (member && !isBuiltIn($.program, member)) {
    sources.push(member);
  }
  if (!isBuiltIn($.program, type)) {
    sources.push(type);
  }

  let doc: string | undefined;
  for (const source of sources) {
    const sourceDoc = $.type.getDoc(source);
    if (sourceDoc) {
      doc = sourceDoc;
      break;
    }
  }

  if (!doc) {
    return [];
  }

  return [callPart("describe", `"${doc.replace(/\n+/g, " ").replace(/"/g, '\\"')}"`)];
}
