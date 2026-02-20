import { SearchFilter } from "../../bindings/hamster-wheel/internal/db/models";
import { listSourceOptions } from "./jobSource";

export interface FilterGroup {
  ID: string;
  Name: string;
  Keywords: string;
  Location: string;
  Sources: string[];
  FilterIDs: string[];
  Filters: SearchFilter[];
  Enabled: boolean;
  AllEnabled: boolean;
  EnabledSourceCount: number;
}

interface FilterGroupAccumulator {
  ID: string;
  Name: string;
  Keywords: string;
  Location: string;
  Sources: Set<string>;
  FilterIDs: string[];
  Filters: SearchFilter[];
  EnabledSourceCount: number;
}

const GROUP_ID_PREFIX = "group:";
const GROUP_KEY_SEPARATOR = "\u001f";

function buildGroupKey(name: string, keywords: string, location: string): string {
  return [name, keywords, location].join(GROUP_KEY_SEPARATOR);
}

export function buildFilterGroupID(
  name: string,
  keywords: string,
  location: string
): string {
  return `${GROUP_ID_PREFIX}${encodeURIComponent(
    buildGroupKey(name, keywords, location)
  )}`;
}

export function groupFilters(filters: SearchFilter[]): FilterGroup[] {
  const grouped = new Map<string, FilterGroupAccumulator>();

  for (const filter of filters) {
    const key = buildGroupKey(filter.Name, filter.Keywords, filter.Location);
    let group = grouped.get(key);
    if (!group) {
      group = {
        ID: buildFilterGroupID(filter.Name, filter.Keywords, filter.Location),
        Name: filter.Name,
        Keywords: filter.Keywords,
        Location: filter.Location,
        Sources: new Set<string>(),
        FilterIDs: [],
        Filters: [],
        EnabledSourceCount: 0,
      };
      grouped.set(key, group);
    }

    group.FilterIDs.push(filter.ID);
    group.Filters.push(filter);
    group.Sources.add(filter.Source);
    if (filter.Enabled) {
      group.EnabledSourceCount += 1;
    }
  }

  const sourceOrder = new Map(
    listSourceOptions().map((option, index) => [option.value, index])
  );

  return Array.from(grouped.values()).map((group) => {
    const sources = Array.from(group.Sources).sort((left, right) => {
      const leftOrder = sourceOrder.get(left) ?? Number.MAX_SAFE_INTEGER;
      const rightOrder = sourceOrder.get(right) ?? Number.MAX_SAFE_INTEGER;
      if (leftOrder !== rightOrder) {
        return leftOrder - rightOrder;
      }
      return left.localeCompare(right);
    });

    const sourceCount = group.FilterIDs.length;
    const enabledSourceCount = group.EnabledSourceCount;
    return {
      ID: group.ID,
      Name: group.Name,
      Keywords: group.Keywords,
      Location: group.Location,
      Sources: sources,
      FilterIDs: group.FilterIDs,
      Filters: group.Filters,
      Enabled: enabledSourceCount > 0,
      AllEnabled: sourceCount > 0 && enabledSourceCount === sourceCount,
      EnabledSourceCount: enabledSourceCount,
    };
  });
}

export function resolveFilterGroupID(
  filterGroups: FilterGroup[],
  filterIDOrGroupID: string | null
): string | null {
  if (!filterIDOrGroupID) {
    return null;
  }

  if (filterGroups.some((group) => group.ID === filterIDOrGroupID)) {
    return filterIDOrGroupID;
  }

  const legacyGroup = filterGroups.find((group) =>
    group.FilterIDs.includes(filterIDOrGroupID)
  );
  return legacyGroup?.ID ?? null;
}
