"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth-context";
import { useWorkspace } from "@/lib/workspace-context";
import { useLang } from "@/lib/lang-context";
import { t, errorMessage } from "@/lib/i18n";
import type { Lang } from "@/lib/i18n";
import { ApiError } from "@/lib/api";
import { getCurrentWorkspace } from "@/lib/workspace-api";
import {
  listPositions,
  createPosition,
  updatePosition,
  deprecatePosition,
} from "@/lib/position-api";
import type { PositionKind } from "@/lib/position-api";
import type { Position } from "@/lib/types";
import {
  positionStatusColor,
  displayPositionLabel,
  sortPositions,
} from "@/lib/positions-masters";
import {
  Input,
  Button,
  Alert,
  Select,
  Modal,
  ConfirmDialog,
  Badge,
  Breadcrumb,
} from "@/components/ui";

// Mirrors the BE CHECK constraint ck_*_positions_code (migration 000014).
const CODE_RE = /^[a-z][a-z0-9_]{1,62}$/;

const inputClass =
  "rounded-md border px-3 py-2 text-sm outline-none transition focus:ring-2 focus:ring-brand-500/40 border-slate-300 bg-white focus:border-brand-500";

export default function PositionsSettingsPage() {
  const { status } = useAuth();
  const { activeSlug, clearActiveSlug } = useWorkspace();
  const { lang } = useLang();
  const router = useRouter();

  const [canMutate, setCanMutate] = useState(false);

  // Per-kind list state
  const [projectList, setProjectList] = useState<Position[]>([]);
  const [companyList, setCompanyList] = useState<Position[]>([]);
  const [projectLoading, setProjectLoading] = useState(true);
  const [companyLoading, setCompanyLoading] = useState(true);
  const [projectError, setProjectError] = useState<string | null>(null);
  const [companyError, setCompanyError] = useState<string | null>(null);

  const [actionError, setActionError] = useState<string | null>(null);

  // Create / Edit modal
  const [modalOpen, setModalOpen] = useState(false);
  const [modalKind, setModalKind] = useState<PositionKind>("project");
  const [modalMode, setModalMode] = useState<"create" | "edit">("create");
  const [fCode, setFCode] = useState("");
  const [fLabelTh, setFLabelTh] = useState("");
  const [fLabelEn, setFLabelEn] = useState("");
  const [fDesc, setFDesc] = useState("");
  const [fSort, setFSort] = useState("0");
  const [fStatus, setFStatus] = useState("active");
  const [codeError, setCodeError] = useState<string | null>(null);
  const [labelThError, setLabelThError] = useState<string | null>(null);
  const [labelEnError, setLabelEnError] = useState<string | null>(null);
  const [sortError, setSortError] = useState<string | null>(null);
  const [statusError, setStatusError] = useState<string | null>(null);
  const [formError, setFormError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  // Deprecate dialog
  const [deprecateOpen, setDeprecateOpen] = useState(false);
  const [deprecateKind, setDeprecateKind] = useState<PositionKind>("project");
  const [deprecateTarget, setDeprecateTarget] = useState<Position | null>(null);
  const [deprecating, setDeprecating] = useState(false);

  // Redirect anonymous users
  useEffect(() => {
    if (status === "anonymous") router.push("/login");
  }, [status, router]);

  // Redirect to workspaces when authenticated but no active workspace
  useEffect(() => {
    if (status === "authenticated" && !activeSlug) router.push("/workspaces");
  }, [status, activeSlug, router]);

  // Load org role (canMutate)
  useEffect(() => {
    if (status !== "authenticated" || !activeSlug) return;
    const load = async () => {
      try {
        const ws = await getCurrentWorkspace(activeSlug);
        setCanMutate(ws.org_role_code === "owner" || ws.org_role_code === "admin");
      } catch (err) {
        if (
          err instanceof ApiError &&
          (err.code === "tenant.workspace_not_found" || err.code === "tenant.forbidden")
        ) {
          clearActiveSlug();
          router.push("/workspaces");
        }
        // else: leave canMutate=false; the list effects surface any real error
      }
    };
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [status, activeSlug]);

  // Load both master lists
  useEffect(() => {
    if (status !== "authenticated" || !activeSlug) return;
    void loadList("project");
    void loadList("company");
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [status, activeSlug, lang]);

  // Page through all positions for a kind (this settings page manages active AND
  // deprecated, so no status filter). limit=100 + paging avoids silent truncation.
  async function loadAll(kind: PositionKind): Promise<Position[]> {
    if (!activeSlug) return [];
    const acc: Position[] = [];
    let page = 1;
    for (;;) {
      const res = await listPositions(activeSlug, kind, { page, limit: 100 });
      acc.push(...res.items);
      if (!res.pagination.has_more) break;
      page += 1;
    }
    return acc;
  }

  async function loadList(kind: PositionKind) {
    const setLoading = kind === "project" ? setProjectLoading : setCompanyLoading;
    const setList = kind === "project" ? setProjectList : setCompanyList;
    const setErr = kind === "project" ? setProjectError : setCompanyError;
    setLoading(true);
    setErr(null);
    try {
      const items = await loadAll(kind);
      setList(items);
    } catch (err) {
      if (
        err instanceof ApiError &&
        (err.code === "tenant.workspace_not_found" || err.code === "tenant.forbidden")
      ) {
        clearActiveSlug();
        router.push("/workspaces");
      } else {
        setErr(
          err instanceof ApiError
            ? errorMessage(err.code, lang)
            : errorMessage("UNKNOWN_ERROR", lang)
        );
      }
    } finally {
      setLoading(false);
    }
  }

  function resetForm() {
    setFCode("");
    setFLabelTh("");
    setFLabelEn("");
    setFDesc("");
    setFSort("0");
    setFStatus("active");
    setCodeError(null);
    setLabelThError(null);
    setLabelEnError(null);
    setSortError(null);
    setStatusError(null);
    setFormError(null);
  }

  function openCreate(kind: PositionKind) {
    resetForm();
    setModalKind(kind);
    setModalMode("create");
    setModalOpen(true);
  }

  function openEdit(kind: PositionKind, p: Position) {
    resetForm();
    setModalKind(kind);
    setModalMode("edit");
    setFCode(p.code);
    setFLabelTh(p.label_th);
    setFLabelEn(p.label_en);
    setFDesc(p.description ?? "");
    setFSort(String(p.sort_order));
    setFStatus(p.status);
    setModalOpen(true);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!activeSlug) return;
    setCodeError(null);
    setLabelThError(null);
    setLabelEnError(null);
    setSortError(null);
    setStatusError(null);
    setFormError(null);

    // Client-side validation (mirrors BE rules so errors are caught pre-submit).
    let bad = false;
    const code = fCode.trim();
    if (modalMode === "create") {
      if (!code) {
        setCodeError(t("position_code_required", lang));
        bad = true;
      } else if (!CODE_RE.test(code)) {
        setCodeError(t("position_code_invalid", lang));
        bad = true;
      }
    }
    const labelTh = fLabelTh.trim();
    const labelEn = fLabelEn.trim();
    if (!labelTh) {
      setLabelThError(t("position_label_required", lang));
      bad = true;
    } else if (labelTh.length > 200) {
      setLabelThError(t("position_label_too_long", lang));
      bad = true;
    }
    if (!labelEn) {
      setLabelEnError(t("position_label_required", lang));
      bad = true;
    } else if (labelEn.length > 200) {
      setLabelEnError(t("position_label_too_long", lang));
      bad = true;
    }
    const sortNum = Number(fSort);
    if (fSort.trim() === "" || !Number.isInteger(sortNum) || sortNum < 0) {
      setSortError(t("position_sort_order_invalid", lang));
      bad = true;
    }
    if (bad) return;

    const desc = fDesc.trim();
    setSaving(true);
    try {
      if (modalMode === "create") {
        await createPosition(activeSlug, modalKind, {
          code,
          label_th: labelTh,
          label_en: labelEn,
          description: desc || undefined,
          sort_order: sortNum,
        });
      } else {
        await updatePosition(activeSlug, modalKind, fCode, {
          code: fCode,
          label_th: labelTh,
          label_en: labelEn,
          description: desc || undefined,
          sort_order: sortNum,
          status: fStatus,
        });
      }
      setModalOpen(false);
      await loadList(modalKind);
    } catch (err) {
      handleSubmitError(err);
    } finally {
      setSaving(false);
    }
  }

  function handleSubmitError(err: unknown) {
    if (!(err instanceof ApiError)) {
      setFormError(errorMessage("UNKNOWN_ERROR", lang));
      return;
    }
    if (err.code === "validation.invalid_input") {
      const fields = (
        err.details as unknown as { fields?: { field: string; message: string }[] } | undefined
      )?.fields;
      let mapped = false;
      if (Array.isArray(fields)) {
        for (const d of fields) {
          if (d.field === "code") {
            setCodeError(t("position_code_invalid", lang));
            mapped = true;
          } else if (d.field === "label_th") {
            setLabelThError(t("position_label_required", lang));
            mapped = true;
          } else if (d.field === "label_en") {
            setLabelEnError(t("position_label_required", lang));
            mapped = true;
          } else if (d.field === "sort_order") {
            setSortError(t("position_sort_order_invalid", lang));
            mapped = true;
          } else if (d.field === "description") {
            setFormError(t("description_too_long", lang));
            mapped = true;
          }
        }
      }
      if (!mapped) setFormError(errorMessage("validation.invalid_input", lang));
    } else if (err.code.endsWith(".code_taken") || err.code.endsWith(".code_immutable")) {
      setCodeError(errorMessage(err.code, lang));
    } else if (err.code.endsWith(".invalid_status")) {
      setStatusError(errorMessage(err.code, lang));
    } else if (err.code === "tenant.workspace_not_found" || err.code === "tenant.forbidden") {
      clearActiveSlug();
      router.push("/workspaces");
    } else {
      setFormError(errorMessage(err.code, lang));
    }
  }

  function openDeprecate(kind: PositionKind, p: Position) {
    setDeprecateKind(kind);
    setDeprecateTarget(p);
    setDeprecateOpen(true);
  }

  async function confirmDeprecate() {
    if (!activeSlug || !deprecateTarget) return;
    setActionError(null);
    setDeprecating(true);
    try {
      await deprecatePosition(activeSlug, deprecateKind, deprecateTarget.code);
      setDeprecateOpen(false);
      setDeprecateTarget(null);
      await loadList(deprecateKind);
    } catch (err) {
      setDeprecateOpen(false);
      setDeprecateTarget(null);
      if (
        err instanceof ApiError &&
        (err.code === "tenant.workspace_not_found" || err.code === "tenant.forbidden")
      ) {
        clearActiveSlug();
        router.push("/workspaces");
      } else {
        setActionError(
          err instanceof ApiError
            ? errorMessage(err.code, lang)
            : errorMessage("UNKNOWN_ERROR", lang)
        );
      }
    } finally {
      setDeprecating(false);
    }
  }

  if (status === "loading") {
    return (
      <div className="flex-1 flex items-center justify-center text-slate-500 text-sm">
        {t("msg.loading", lang)}
      </div>
    );
  }
  if (status === "anonymous" || !activeSlug) return null;

  return (
    <div className="flex-1 p-8 max-w-2xl mx-auto w-full flex flex-col gap-8">
      <div className="flex flex-col gap-2">
        <Breadcrumb
          items={[
            { label: t("nav.settings", lang) },
            { label: t("page.positions.title", lang) },
          ]}
        />
        <h1 className="text-2xl font-semibold text-slate-800">{t("page.positions.title", lang)}</h1>
      </div>

      {actionError && <Alert variant="error">{actionError}</Alert>}

      <MasterCard
        heading={t("positions.project_heading", lang)}
        desc={t("positions.project_desc", lang)}
        items={projectList}
        loading={projectLoading}
        error={projectError}
        canMutate={canMutate}
        lang={lang}
        onAdd={() => openCreate("project")}
        onEdit={(p) => openEdit("project", p)}
        onDeprecate={(p) => openDeprecate("project", p)}
      />
      <MasterCard
        heading={t("positions.company_heading", lang)}
        desc={t("positions.company_desc", lang)}
        items={companyList}
        loading={companyLoading}
        error={companyError}
        canMutate={canMutate}
        lang={lang}
        onAdd={() => openCreate("company")}
        onEdit={(p) => openEdit("company", p)}
        onDeprecate={(p) => openDeprecate("company", p)}
      />

      {/* Create / Edit modal (shared; mode toggled by state) */}
      <Modal
        open={modalOpen}
        title={t(
          modalMode === "create" ? "dialog.create_position.title" : "dialog.edit_position.title",
          lang
        )}
        onClose={() => setModalOpen(false)}
        className="w-full max-w-md max-h-[85vh] overflow-y-auto"
      >
        {formError && <Alert variant="error">{formError}</Alert>}
        <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
          <Input
            id="position-code"
            label={t("label.position_code", lang)}
            value={fCode}
            onChange={(e) => setFCode(e.target.value)}
            error={codeError ?? undefined}
            disabled={saving || modalMode === "edit"}
            autoComplete="off"
            placeholder="lead_dev"
          />
          <Input
            id="position-label-th"
            label={t("label.position_label_th", lang)}
            value={fLabelTh}
            onChange={(e) => setFLabelTh(e.target.value)}
            error={labelThError ?? undefined}
            disabled={saving}
            autoComplete="off"
          />
          <Input
            id="position-label-en"
            label={t("label.position_label_en", lang)}
            value={fLabelEn}
            onChange={(e) => setFLabelEn(e.target.value)}
            error={labelEnError ?? undefined}
            disabled={saving}
            autoComplete="off"
          />
          <div className="flex flex-col gap-1">
            <label htmlFor="position-description" className="text-sm font-medium text-slate-700">
              {t("label.position_description", lang)}
            </label>
            <textarea
              id="position-description"
              className={inputClass}
              value={fDesc}
              onChange={(e) => setFDesc(e.target.value)}
              disabled={saving}
              rows={3}
            />
          </div>
          <Input
            id="position-sort-order"
            type="number"
            label={t("label.position_sort_order", lang)}
            value={fSort}
            onChange={(e) => setFSort(e.target.value)}
            error={sortError ?? undefined}
            disabled={saving}
            min={0}
          />
          {modalMode === "edit" && (
            <Select
              id="position-status"
              label={t("label.position_status", lang)}
              value={fStatus}
              options={[
                { value: "active", label: t("status.position.active", lang) },
                { value: "deprecated", label: t("status.position.deprecated", lang) },
              ]}
              onChange={(e) => setFStatus(e.target.value)}
              error={statusError ?? undefined}
              disabled={saving}
            />
          )}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="ghost" onClick={() => setModalOpen(false)} disabled={saving}>
              {t("btn.cancel", lang)}
            </Button>
            <Button type="submit" loading={saving}>
              {t("btn.save", lang)}
            </Button>
          </div>
        </form>
      </Modal>

      <ConfirmDialog
        open={deprecateOpen}
        title={t("dialog.deprecate_position.title", lang)}
        body={t("dialog.deprecate_position.body", lang)}
        confirmLabel={t("btn.deprecate", lang)}
        cancelLabel={t("btn.cancel", lang)}
        confirmVariant="danger"
        loading={deprecating}
        onConfirm={confirmDeprecate}
        onCancel={() => {
          setDeprecateOpen(false);
          setDeprecateTarget(null);
        }}
      />
    </div>
  );
}

interface MasterCardProps {
  heading: string;
  desc: string;
  items: Position[];
  loading: boolean;
  error: string | null;
  canMutate: boolean;
  lang: Lang;
  onAdd(): void;
  onEdit(p: Position): void;
  onDeprecate(p: Position): void;
}

function MasterCard({
  heading,
  desc,
  items,
  loading,
  error,
  canMutate,
  lang,
  onAdd,
  onEdit,
  onDeprecate,
}: MasterCardProps) {
  const sorted = sortPositions(items);
  return (
    <div className="bg-white rounded-2xl shadow-sm border border-slate-200 p-6 flex flex-col gap-4">
      <div className="flex items-start justify-between gap-4">
        <div className="flex flex-col gap-1">
          <h2 className="text-lg font-semibold text-slate-800">{heading}</h2>
          <p className="text-sm text-slate-500">{desc}</p>
        </div>
        {canMutate && (
          <Button variant="secondary" onClick={onAdd}>
            {t("positions.add", lang)}
          </Button>
        )}
      </div>

      {error && <Alert variant="error">{error}</Alert>}

      {loading ? (
        <p className="text-sm text-slate-500">{t("msg.loading", lang)}</p>
      ) : sorted.length === 0 ? (
        <div className="flex flex-col items-center gap-3 rounded-xl border border-dashed border-slate-200 px-6 py-8 text-center">
          <p className="text-sm text-slate-500">
            {canMutate ? t("positions.empty", lang) : t("msg.positions_readonly", lang)}
          </p>
          {canMutate && (
            <Button variant="secondary" onClick={onAdd}>
              {t("positions.add", lang)}
            </Button>
          )}
        </div>
      ) : (
        <ul className="flex flex-col divide-y divide-slate-100">
          {sorted.map((p) => (
            <li
              key={p.id}
              className="flex flex-wrap items-center justify-between gap-3 py-3 first:pt-0 last:pb-0"
            >
              <div className="flex items-center gap-2 min-w-0">
                <span className="text-sm text-slate-800 truncate">{displayPositionLabel(p, lang)}</span>
                <span className="text-xs text-slate-400 truncate">
                  {lang === "th" ? p.label_en : p.label_th}
                </span>
                <span className="font-mono text-xs text-slate-400">{p.code}</span>
                {p.status === "deprecated" && (
                  <Badge color={positionStatusColor("deprecated")}>
                    {t("positions.deprecated_badge", lang)}
                  </Badge>
                )}
                {p.is_system && <Badge>{t("positions.system_badge", lang)}</Badge>}
              </div>
              {canMutate && (
                <div className="flex items-center gap-2">
                  {!p.is_system && (
                    <Button variant="ghost" onClick={() => onEdit(p)}>
                      {t("btn.edit", lang)}
                    </Button>
                  )}
                  {p.status === "active" && !p.is_system && (
                    <Button variant="danger" onClick={() => onDeprecate(p)}>
                      {t("btn.deprecate", lang)}
                    </Button>
                  )}
                </div>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
