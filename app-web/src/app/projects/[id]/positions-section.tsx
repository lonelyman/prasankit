"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useLang } from "@/lib/lang-context";
import { t, errorMessage } from "@/lib/i18n";
import { ApiError } from "@/lib/api";
import {
  listPositions,
  listMemberPositions,
  assignMemberPosition,
  unassignMemberPosition,
  setCompanyPosition,
} from "@/lib/position-api";
import type { Position, ProjectMember, ProjectMemberPosition } from "@/lib/types";
import { displayPositionLabel, positionStatusColor, sortPositions } from "@/lib/positions-masters";
import { Button, Alert, Select, Modal, ConfirmDialog, Badge } from "@/components/ui";
import type { SelectOption } from "@/components/ui";

interface PositionsSectionProps {
  slug: string;
  projectId: string;
  members: ProjectMember[];
  membersLoading: boolean;
  canMutate: boolean;
  onReconcile(): void; // page's loadMembers — refetch members (company_position_code re-sync / stale member)
  onTenantError(): void; // page clears the active workspace + redirects to /workspaces
}

export function PositionsSection({
  slug,
  projectId,
  members,
  membersLoading,
  canMutate,
  onReconcile,
  onTenantError,
}: PositionsSectionProps) {
  const { lang } = useLang();

  const [projectMasters, setProjectMasters] = useState<Position[]>([]);
  const [companyMasters, setCompanyMasters] = useState<Position[]>([]);
  const [mastersLoading, setMastersLoading] = useState(true);
  const [mastersError, setMastersError] = useState<string | null>(null);

  const [memberPositions, setMemberPositions] = useState<Record<string, ProjectMemberPosition[]>>({});
  const [sectionError, setSectionError] = useState<string | null>(null);
  const [companySavingId, setCompanySavingId] = useState<string | null>(null);
  // Optimistic company-position value per membership: the Select is otherwise bound
  // to the (async-refetched) prop and would snap back to the old value mid-save.
  const [companyValue, setCompanyValue] = useState<Record<string, string>>({});

  // Assign modal
  const [assignOpen, setAssignOpen] = useState(false);
  const [assignMember, setAssignMember] = useState<ProjectMember | null>(null);
  const [assignCode, setAssignCode] = useState("");
  const [assignError, setAssignError] = useState<string | null>(null);
  const [assignFieldError, setAssignFieldError] = useState<string | null>(null);
  const [assigning, setAssigning] = useState(false);

  // Unassign dialog
  const [unassignTarget, setUnassignTarget] = useState<{ member: ProjectMember; position: ProjectMemberPosition } | null>(null);
  const [unassigning, setUnassigning] = useState(false);

  // ── loaders ────────────────────────────────────────────────────────────────

  // Page through all active positions for a kind (picker shows active only).
  async function loadActiveMasters(kind: "project" | "company"): Promise<Position[]> {
    const acc: Position[] = [];
    let page = 1;
    for (;;) {
      const res = await listPositions(slug, kind, { status: "active", page, limit: 100 });
      acc.push(...res.items);
      if (!res.pagination.has_more) break;
      page += 1;
    }
    return acc;
  }

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      setMastersLoading(true);
      setMastersError(null);
      try {
        const [proj, comp] = await Promise.all([
          loadActiveMasters("project"),
          loadActiveMasters("company"),
        ]);
        if (cancelled) return;
        setProjectMasters(proj);
        setCompanyMasters(comp);
      } catch (err) {
        if (cancelled) return;
        if (
          err instanceof ApiError &&
          (err.code === "tenant.workspace_not_found" || err.code === "tenant.forbidden")
        ) {
          onTenantError();
        } else {
          setMastersError(err instanceof ApiError ? errorMessage(err.code, lang) : errorMessage("UNKNOWN_ERROR", lang));
        }
      } finally {
        if (!cancelled) setMastersLoading(false);
      }
    };
    void load();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [slug]);

  // Reload member positions whenever the SET of member ids changes (not on every
  // members re-fetch / lang switch — the membersKey is stable across those).
  const membersKey = members.map((m) => m.id).join(",");
  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      if (members.length === 0) {
        setMemberPositions({});
        return;
      }
      const entries = await Promise.all(
        members.map(async (m) => {
          try {
            const r = await listMemberPositions(slug, projectId, m.id);
            return [m.id, r.items] as const;
          } catch {
            // A member could be removed mid-session; default to empty rather than
            // failing the whole section. A persistent issue surfaces on mutate.
            return [m.id, [] as ProjectMemberPosition[]] as const;
          }
        })
      );
      if (!cancelled) setMemberPositions(Object.fromEntries(entries));
    };
    void load();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [slug, projectId, membersKey]);

  async function reloadMember(memberId: string) {
    try {
      const r = await listMemberPositions(slug, projectId, memberId);
      setMemberPositions((prev) => ({ ...prev, [memberId]: r.items }));
    } catch {
      // leave as-is; the next full reconcile covers a persistent error
    }
  }

  // ── error routing ────────────────────────────────────────────────────────────

  function handlePosError(err: unknown, setErr: (m: string) => void) {
    if (!(err instanceof ApiError)) {
      setErr(errorMessage("UNKNOWN_ERROR", lang));
      return;
    }
    if (err.code === "tenant.workspace_not_found" || err.code === "tenant.forbidden") {
      onTenantError();
      return;
    }
    setErr(errorMessage(err.code, lang));
    // Stale member/membership → re-sync the members list (page-owned).
    if (
      err.code === "project_member_position.member_removed" ||
      err.code === "project_member_position.member_not_found" ||
      err.code === "company_position.membership_not_found" ||
      err.code === "project.not_found"
    ) {
      onReconcile();
    }
  }

  // ── handlers ───────────────────────────────────────────────────────────────

  function assignedCodes(memberId: string): Set<string> {
    return new Set((memberPositions[memberId] ?? []).map((p) => p.project_position_code));
  }

  // Active project masters this member does NOT already hold.
  function addableMasters(memberId: string): Position[] {
    const have = assignedCodes(memberId);
    return sortPositions(projectMasters.filter((pm) => !have.has(pm.code)));
  }

  function openAssign(member: ProjectMember) {
    setAssignMember(member);
    setAssignCode("");
    setAssignError(null);
    setAssignFieldError(null);
    setAssignOpen(true);
  }

  function closeAssign() {
    setAssignOpen(false);
    setAssignMember(null);
  }

  async function handleAssign(e: React.FormEvent) {
    e.preventDefault();
    if (!assignMember) return;
    setAssignError(null);
    setAssignFieldError(null);
    if (!assignCode) {
      setAssignFieldError(t("label.select_position", lang));
      return;
    }
    setAssigning(true);
    try {
      await assignMemberPosition(slug, projectId, assignMember.id, assignCode);
      const memberId = assignMember.id;
      closeAssign();
      await reloadMember(memberId);
    } catch (err) {
      handlePosError(err, setAssignError);
    } finally {
      setAssigning(false);
    }
  }

  function openUnassign(member: ProjectMember, position: ProjectMemberPosition) {
    setUnassignTarget({ member, position });
  }

  async function confirmUnassign() {
    if (!unassignTarget) return;
    const { member, position } = unassignTarget;
    setSectionError(null);
    setUnassigning(true);
    try {
      await unassignMemberPosition(slug, projectId, member.id, position.project_position_code);
      setUnassignTarget(null);
      await reloadMember(member.id);
    } catch (err) {
      setUnassignTarget(null);
      handlePosError(err, setSectionError);
    } finally {
      setUnassigning(false);
    }
  }

  async function onCompanyChange(member: ProjectMember, value: string) {
    const membershipId = member.workspace_membership_id;
    setSectionError(null);
    setCompanySavingId(membershipId);
    setCompanyValue((prev) => ({ ...prev, [membershipId]: value })); // optimistic
    try {
      await setCompanyPosition(slug, membershipId, value || null);
      onReconcile(); // refetch members so company_position_code re-syncs (page-canonical)
    } catch (err) {
      setCompanyValue((prev) => ({ ...prev, [membershipId]: member.company_position_code ?? "" })); // revert
      handlePosError(err, setSectionError);
    } finally {
      setCompanySavingId(null);
    }
  }

  // ── derived ──────────────────────────────────────────────────────────────────

  const noActiveProjectMasters = !mastersLoading && projectMasters.length === 0;

  function companyLabel(code: string): string {
    const cp = companyMasters.find((c) => c.code === code);
    return cp ? displayPositionLabel(cp, lang) : code;
  }

  function companyOptions(member: ProjectMember): SelectOption[] {
    const opts: SelectOption[] = [{ value: "", label: t("positions.company_position_none", lang) }];
    const seen = new Set<string>();
    for (const cp of sortPositions(companyMasters)) {
      opts.push({ value: cp.code, label: displayPositionLabel(cp, lang) });
      seen.add(cp.code);
    }
    // Keep a deprecated current value visible (it is not in the active list).
    const cur = member.company_position_code;
    if (cur && !seen.has(cur)) opts.push({ value: cur, label: cur });
    return opts;
  }

  // ── render ─────────────────────────────────────────────────────────────────

  return (
    <div className="bg-white rounded-2xl shadow-sm border border-slate-200 p-6 flex flex-col gap-4">
      <h2 className="text-lg font-semibold text-slate-800">{t("positions.member_heading", lang)}</h2>
      <p className="text-xs text-slate-400">{t("positions.company_position_hint", lang)}</p>

      {sectionError && <Alert variant="error">{sectionError}</Alert>}
      {mastersError && <Alert variant="error">{mastersError}</Alert>}
      {noActiveProjectMasters && canMutate && (
        <Alert variant="info">
          {t("positions.no_active_masters", lang)}{" "}
          <Link href="/settings/positions" className="font-medium underline underline-offset-2">
            {t("positions.go_to_settings", lang)}
          </Link>
        </Alert>
      )}

      {mastersLoading || (membersLoading && members.length === 0) ? (
        // Show loading only on the initial load — keep rendering existing rows during
        // a post-mutate refetch (company-position change refetches members) so the
        // section does not blink to "loading".
        <p className="text-sm text-slate-500">{t("msg.loading", lang)}</p>
      ) : members.length === 0 ? (
        <p className="text-sm text-slate-500">{t("team.empty", lang)}</p>
      ) : (
        <ul className="flex flex-col divide-y divide-slate-100">
          {members.map((m) => {
            const assigned = memberPositions[m.id] ?? [];
            const canAdd = canMutate && addableMasters(m.id).length > 0;
            const companySaving = companySavingId === m.workspace_membership_id;
            return (
              <li key={m.id} className="flex flex-col gap-2 py-3 first:pt-0 last:pb-0">
                <span className="text-sm font-medium text-slate-800 truncate">{m.display_name ?? "—"}</span>

                {/* project positions (M:N) */}
                <div className="flex flex-wrap items-center gap-2">
                  <span className="w-28 shrink-0 text-xs text-slate-400">
                    {t("positions.project_heading", lang)}
                  </span>
                  {assigned.length === 0 && (
                    <span className="text-xs text-slate-400">{t("positions.member_none", lang)}</span>
                  )}
                  {assigned.map((p) => {
                    const deprecated = p.status === "deprecated";
                    return (
                      <Badge
                        key={p.id}
                        color={deprecated ? positionStatusColor("deprecated") : "bg-brand-50 text-brand-700"}
                      >
                        {lang === "th" ? p.label_th : p.label_en}
                        {canMutate && (
                          <button
                            type="button"
                            onClick={() => openUnassign(m, p)}
                            aria-label={t("btn.remove", lang)}
                            className="ml-1 -mr-0.5 rounded-full px-1 leading-none hover:bg-black/10 cursor-pointer"
                          >
                            ×
                          </button>
                        )}
                      </Badge>
                    );
                  })}
                  {canAdd && (
                    <Button variant="ghost" className="px-2 py-1 text-xs" onClick={() => openAssign(m)}>
                      {t("positions.add", lang)}
                    </Button>
                  )}
                </div>

                {/* company position (1:1, workspace-level) */}
                <div className="flex flex-wrap items-center gap-2">
                  <span className="w-28 shrink-0 text-xs text-slate-400">
                    {t("positions.company_position_label", lang)}
                  </span>
                  {canMutate ? (
                    <Select
                      id={`company-${m.id}`}
                      aria-label={t("positions.company_position_label", lang)}
                      className="min-w-[12rem]"
                      value={companyValue[m.workspace_membership_id] ?? (m.company_position_code ?? "")}
                      options={companyOptions(m)}
                      onChange={(e) => onCompanyChange(m, e.target.value)}
                      disabled={companySaving}
                    />
                  ) : m.company_position_code ? (
                    <Badge>{companyLabel(m.company_position_code)}</Badge>
                  ) : (
                    <span className="text-xs text-slate-400">{t("positions.company_position_none", lang)}</span>
                  )}
                </div>
              </li>
            );
          })}
        </ul>
      )}

      {/* Assign project position modal */}
      <Modal open={assignOpen} title={t("dialog.assign_position.title", lang)} onClose={closeAssign}>
        {assignError && <Alert variant="error">{assignError}</Alert>}
        <form onSubmit={handleAssign} className="flex flex-col gap-4" noValidate>
          <Select
            id="assign-position-select"
            label={t("label.select_position", lang)}
            value={assignCode}
            options={[
              { value: "", label: t("label.select_position", lang) },
              ...(assignMember ? addableMasters(assignMember.id) : []).map((pm) => ({
                value: pm.code,
                label: displayPositionLabel(pm, lang),
              })),
            ]}
            onChange={(e) => setAssignCode(e.target.value)}
            error={assignFieldError ?? undefined}
            disabled={assigning}
          />
          <div className="flex justify-end gap-2">
            <Button type="button" variant="ghost" onClick={closeAssign} disabled={assigning}>
              {t("btn.cancel", lang)}
            </Button>
            <Button type="submit" loading={assigning}>
              {t("btn.add", lang)}
            </Button>
          </div>
        </form>
      </Modal>

      <ConfirmDialog
        open={unassignTarget !== null}
        title={t("dialog.unassign_position.title", lang)}
        body={t("dialog.unassign_position.body", lang)}
        confirmLabel={t("btn.remove", lang)}
        cancelLabel={t("btn.cancel", lang)}
        confirmVariant="danger"
        loading={unassigning}
        onConfirm={confirmUnassign}
        onCancel={() => setUnassignTarget(null)}
      />
    </div>
  );
}
