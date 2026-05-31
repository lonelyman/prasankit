"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth-context";
import { useWorkspace } from "@/lib/workspace-context";
import { useLang } from "@/lib/lang-context";
import { t, errorMessage } from "@/lib/i18n";
import {
  validateProjectName,
  validateProjectSlugOptional,
  validateRequestingUnit,
  validateDescription,
  validateProjectDate,
  validateDateRange,
} from "@/lib/validation";
import { getCurrentWorkspace } from "@/lib/workspace-api";
import { listProjects, createProject } from "@/lib/project-api";
import type { CreateProjectInput } from "@/lib/project-api";
import { ApiError } from "@/lib/api";
import type { Project, ProjectPagination } from "@/lib/types";
import {
  statusOptions,
  typeOptions,
  statusFilterOptions,
  typeFilterOptions,
} from "@/lib/project-masters";
import { Input, Button, Alert, Select, Pagination } from "@/components/ui";

// Field-name map: BE field → local validation error key + setter (key design call #7).
type FieldKey = "name" | "slug" | "requestingUnit" | "description" | "startDate" | "endDate";
const BE_FIELD_MAP: Record<string, FieldKey> = {
  project_name: "name",
  slug: "slug",
  requesting_unit: "requestingUnit",
  description: "description",
  start_date: "startDate",
  end_date: "endDate",
};

export default function ProjectsPage() {
  const { status } = useAuth();
  const { activeSlug, clearActiveSlug } = useWorkspace();
  const { lang } = useLang();
  const router = useRouter();

  // List state
  const [items, setItems] = useState<Project[]>([]);
  const [pagination, setPagination] = useState<ProjectPagination | null>(null);
  const [listLoading, setListLoading] = useState(true);
  const [listError, setListError] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState("");
  const [typeFilter, setTypeFilter] = useState("");
  const [page, setPage] = useState(1);

  // Org-role state
  const [canMutate, setCanMutate] = useState(false);

  // Create form state
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [projectType, setProjectType] = useState("internal");
  const [projectStatus, setProjectStatus] = useState("draft");
  const [requestingUnit, setRequestingUnit] = useState("");
  const [description, setDescription] = useState("");
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");
  const [nameError, setNameError] = useState<string | null>(null);
  const [slugError, setSlugError] = useState<string | null>(null);
  const [requestingUnitError, setRequestingUnitError] = useState<string | null>(null);
  const [descriptionError, setDescriptionError] = useState<string | null>(null);
  const [startDateError, setStartDateError] = useState<string | null>(null);
  const [endDateError, setEndDateError] = useState<string | null>(null);
  const [typeError, setTypeError] = useState<string | null>(null);
  const [statusError, setStatusError] = useState<string | null>(null);
  const [formError, setFormError] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);

  // Redirect anonymous users
  useEffect(() => {
    if (status === "anonymous") {
      router.push("/login");
    }
  }, [status, router]);

  // Redirect to workspaces when authenticated but no active workspace
  useEffect(() => {
    if (status === "authenticated" && !activeSlug) {
      router.push("/workspaces");
    }
  }, [status, activeSlug, router]);

  // Load current workspace (org role)
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
        // else: leave canMutate=false; the list effect surfaces any real error
      }
    };

    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [status, activeSlug]);

  // Load project list
  useEffect(() => {
    if (status !== "authenticated" || !activeSlug) return;

    const load = async () => {
      setListLoading(true);
      setListError(null);
      try {
        const result = await listProjects(activeSlug, {
          page,
          project_status_code: statusFilter || undefined,
          project_type_code: typeFilter || undefined,
        });
        setItems(result.items);
        setPagination(result.pagination);
      } catch (err) {
        if (
          err instanceof ApiError &&
          (err.code === "tenant.workspace_not_found" || err.code === "tenant.forbidden")
        ) {
          clearActiveSlug();
          router.push("/workspaces");
        } else {
          setListError(
            err instanceof ApiError
              ? errorMessage(err.code, lang)
              : errorMessage("UNKNOWN_ERROR", lang)
          );
        }
      } finally {
        setListLoading(false);
      }
    };

    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [status, activeSlug, lang, page, statusFilter, typeFilter]);

  if (status === "loading") {
    return (
      <div className="flex-1 flex items-center justify-center text-zinc-500 text-sm">
        {t("msg.loading", lang)}
      </div>
    );
  }

  if (status === "anonymous" || !activeSlug) {
    return null;
  }

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!activeSlug) return;
    setNameError(null);
    setSlugError(null);
    setRequestingUnitError(null);
    setDescriptionError(null);
    setStartDateError(null);
    setEndDateError(null);
    setTypeError(null);
    setStatusError(null);
    setFormError(null);

    const nameErr = validateProjectName(name);
    const slugErr = validateProjectSlugOptional(slug);
    const reqUnitErr = validateRequestingUnit(requestingUnit);
    const descErr = validateDescription(description);
    const startErr = validateProjectDate(startDate);
    const endErr = validateProjectDate(endDate);

    if (nameErr) setNameError(t(nameErr, lang));
    if (slugErr) setSlugError(t(slugErr, lang));
    if (reqUnitErr) setRequestingUnitError(t(reqUnitErr, lang));
    if (descErr) setDescriptionError(t(descErr, lang));
    if (startErr) setStartDateError(t(startErr, lang));
    if (endErr) setEndDateError(t(endErr, lang));

    // Only check the range when BOTH dates are structurally valid.
    let rangeErr: string | null = null;
    if (!startErr && !endErr) {
      rangeErr = validateDateRange(startDate, endDate);
      if (rangeErr) setEndDateError(t(rangeErr, lang));
    }

    if (nameErr || slugErr || reqUnitErr || descErr || startErr || endErr || rangeErr) return;

    const input: CreateProjectInput = {
      project_name: name.trim(),
      project_type_code: projectType,
      project_status_code: projectStatus,
    };
    const slugTrimmed = slug.trim();
    if (slugTrimmed) input.slug = slugTrimmed;
    const reqUnitTrimmed = requestingUnit.trim();
    if (reqUnitTrimmed) input.requesting_unit = reqUnitTrimmed;
    const descTrimmed = description.trim();
    if (descTrimmed) input.description = descTrimmed;
    const startTrimmed = startDate.trim();
    if (startTrimmed) input.start_date = startTrimmed;
    const endTrimmed = endDate.trim();
    if (endTrimmed) input.end_date = endTrimmed;

    setCreating(true);
    try {
      const created = await createProject(activeSlug, input);
      router.push(`/projects/${created.id}`);
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.code === "validation.invalid_input") {
          const fields = (
            err.details as unknown as { fields?: { field: string; message: string }[] } | undefined
          )?.fields;
          if (Array.isArray(fields) && fields.length) {
            let mappedAny = false;
            for (const d of fields) {
              const key = BE_FIELD_MAP[d.field];
              if (!key) continue;
              mappedAny = true;
              // Localize by field (not by echoing BE English message).
              // client already guarantees non-empty + ≤200 chars pre-submit, so a server project_name error is the byte-length (Thai) case
              if (key === "name") setNameError(t("project_name_too_long", lang));
              else if (key === "slug") setSlugError(t("slug_invalid", lang));
              else if (key === "requestingUnit") setRequestingUnitError(t("requesting_unit_too_long", lang));
              else if (key === "description") setDescriptionError(t("description_too_long", lang));
              else if (key === "startDate") setStartDateError(t("invalid_date", lang));
              else if (key === "endDate") setEndDateError(t("invalid_date", lang));
            }
            if (!mappedAny) setFormError(errorMessage("validation.invalid_input", lang));
          } else {
            setFormError(errorMessage("validation.invalid_input", lang));
          }
        } else if (err.code === "project.slug_taken") {
          setSlugError(errorMessage(err.code, lang));
        } else if (err.code === "project.invalid_master_code") {
          const f = (err.details as unknown as { field?: string } | undefined)?.field;
          if (f === "project_status_code") setStatusError(errorMessage(err.code, lang));
          else if (f === "project_type_code") setTypeError(errorMessage(err.code, lang));
          setFormError(errorMessage("project.invalid_master_code", lang));
        } else if (
          err.code === "tenant.workspace_not_found" ||
          err.code === "tenant.forbidden"
        ) {
          clearActiveSlug();
          router.push("/workspaces");
        } else {
          setFormError(errorMessage(err.code ?? "UNKNOWN_ERROR", lang));
        }
      } else {
        setFormError(errorMessage("UNKNOWN_ERROR", lang));
      }
    } finally {
      setCreating(false);
    }
  }

  const isEmpty = !listLoading && items.length === 0;
  const showInternalHint = projectType === "internal" && !requestingUnit.trim();

  const inputClass =
    "rounded-md border px-3 py-2 text-sm outline-none transition focus:ring-2 focus:ring-zinc-400 border-zinc-300 bg-white";

  return (
    <div className="flex-1 p-8 max-w-2xl mx-auto w-full flex flex-col gap-8">
      {/* List */}
      <div className="flex flex-col gap-4">
        <h1 className="text-2xl font-semibold text-zinc-800">
          {isEmpty
            ? t("page.projects.empty_heading", lang)
            : t("page.projects.title", lang)}
        </h1>

        {/* Filter bar */}
        <div className="flex gap-3">
          <Select
            id="filter-status"
            label={t("filter.status", lang)}
            value={statusFilter}
            options={statusFilterOptions(lang)}
            onChange={(e) => {
              setPage(1);
              setStatusFilter(e.target.value);
            }}
          />
          <Select
            id="filter-type"
            label={t("filter.type", lang)}
            value={typeFilter}
            options={typeFilterOptions(lang)}
            onChange={(e) => {
              setPage(1);
              setTypeFilter(e.target.value);
            }}
          />
        </div>

        {listLoading && <p className="text-sm text-zinc-500">{t("msg.loading", lang)}</p>}

        {listError && <Alert variant="error">{listError}</Alert>}

        {!listLoading && items.length > 0 && (
          <ul className="flex flex-col gap-2">
            {items.map((p) => (
              <li
                key={p.id}
                className="flex items-center justify-between rounded-lg border border-zinc-200 bg-white px-4 py-3"
              >
                <div>
                  <p className="font-medium text-zinc-800">{p.project_name}</p>
                  <p className="text-xs text-zinc-500">
                    {t(`status.${p.project_status_code}`, lang)} &middot;{" "}
                    {t(`type.${p.project_type_code}`, lang)} &middot; {p.slug ?? "—"}
                  </p>
                </div>
                <Button
                  variant="secondary"
                  onClick={() => router.push(`/projects/${p.id}`)}
                >
                  {t("btn.open_project", lang)}
                </Button>
              </li>
            ))}
          </ul>
        )}

        {pagination && pagination.total_pages > 1 && (
          <Pagination
            page={pagination.page}
            totalPages={pagination.total_pages}
            hasMore={pagination.has_more}
            onPrev={() => setPage((p) => Math.max(1, p - 1))}
            onNext={() => setPage((p) => p + 1)}
            labelPrev={t("btn.prev", lang)}
            labelNext={t("btn.next", lang)}
            labelStatus={t("pagination.page_of", lang)
              .replace("{page}", String(pagination.page))
              .replace("{total}", String(pagination.total_pages))}
            disabled={listLoading}
          />
        )}

        {isEmpty && (
          <p className="text-sm text-zinc-500">{t("msg.projects_empty_hint", lang)}</p>
        )}
      </div>

      {/* Create form — only for owner/admin */}
      {canMutate && (
        <div className="bg-white rounded-2xl shadow-sm border border-zinc-200 p-6 flex flex-col gap-4">
          <h2 className="text-lg font-semibold text-zinc-800">
            {t("page.projects.create_heading", lang)}
          </h2>

          {formError && <Alert variant="error">{formError}</Alert>}

          <form onSubmit={handleCreate} className="flex flex-col gap-4" noValidate>
            <Input
              id="project-name"
              label={t("label.project_name", lang)}
              value={name}
              onChange={(e) => setName(e.target.value)}
              error={nameError ?? undefined}
              disabled={creating}
              autoComplete="off"
            />
            <Input
              id="project-slug"
              label={t("label.project_slug", lang)}
              value={slug}
              onChange={(e) => setSlug(e.target.value)}
              error={slugError ?? undefined}
              disabled={creating}
              autoComplete="off"
              placeholder="my-project"
            />
            <Select
              id="project-type"
              label={t("label.project_type", lang)}
              value={projectType}
              options={typeOptions(lang)}
              onChange={(e) => setProjectType(e.target.value)}
              error={typeError ?? undefined}
              disabled={creating}
            />
            <Select
              id="project-status"
              label={t("label.project_status", lang)}
              value={projectStatus}
              options={statusOptions(lang)}
              onChange={(e) => setProjectStatus(e.target.value)}
              error={statusError ?? undefined}
              disabled={creating}
            />
            <Input
              id="project-requesting-unit"
              label={t("label.requesting_unit", lang)}
              value={requestingUnit}
              onChange={(e) => setRequestingUnit(e.target.value)}
              error={requestingUnitError ?? undefined}
              disabled={creating}
              autoComplete="off"
            />
            {showInternalHint && (
              <p className="text-xs text-blue-600">{t("hint.internal_requesting_unit", lang)}</p>
            )}
            <div className="flex flex-col gap-1">
              <label htmlFor="project-description" className="text-sm font-medium text-zinc-700">
                {t("label.description", lang)}
              </label>
              <textarea
                id="project-description"
                className={inputClass}
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                disabled={creating}
                rows={4}
              />
              {descriptionError && <p className="text-xs text-red-600">{descriptionError}</p>}
            </div>
            <Input
              id="project-start-date"
              type="date"
              label={t("label.start_date", lang)}
              value={startDate}
              onChange={(e) => setStartDate(e.target.value)}
              error={startDateError ?? undefined}
              disabled={creating}
            />
            <Input
              id="project-end-date"
              type="date"
              label={t("label.end_date", lang)}
              value={endDate}
              onChange={(e) => setEndDate(e.target.value)}
              error={endDateError ?? undefined}
              disabled={creating}
            />
            <Button type="submit" loading={creating}>
              {t("btn.create_project", lang)}
            </Button>
          </form>
        </div>
      )}
    </div>
  );
}
