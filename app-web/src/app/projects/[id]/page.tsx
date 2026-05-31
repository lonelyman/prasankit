"use client";

import { useEffect, useState } from "react";
import { useRouter, useParams } from "next/navigation";
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
import {
  getProject,
  updateProject,
  deleteProject,
  changeProjectStatus,
} from "@/lib/project-api";
import type { UpdateProjectInput } from "@/lib/project-api";
import { ApiError } from "@/lib/api";
import type { Project } from "@/lib/types";
import { statusOptions, typeOptions } from "@/lib/project-masters";
import { Input, Button, Alert, Select, ConfirmDialog } from "@/components/ui";

type FieldKey = "name" | "slug" | "requestingUnit" | "description" | "startDate" | "endDate";
const BE_FIELD_MAP: Record<string, FieldKey> = {
  project_name: "name",
  slug: "slug",
  requesting_unit: "requestingUnit",
  description: "description",
  start_date: "startDate",
  end_date: "endDate",
};

export default function ProjectDetailPage() {
  const { status } = useAuth();
  const { activeSlug, clearActiveSlug } = useWorkspace();
  const { lang } = useLang();
  const router = useRouter();
  const params = useParams();
  const id = typeof params.id === "string" ? params.id : Array.isArray(params.id) ? params.id[0] : "";

  // Load state
  const [project, setProject] = useState<Project | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [notFound, setNotFound] = useState(false);
  const [canMutate, setCanMutate] = useState(false);

  // Edit state
  const [isEditing, setIsEditing] = useState(false);
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [projectType, setProjectType] = useState("internal");
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
  const [formError, setFormError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  // Status-change state
  const [statusDialogOpen, setStatusDialogOpen] = useState(false);
  const [pendingStatus, setPendingStatus] = useState("");
  const [statusSaving, setStatusSaving] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  // Delete state
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);

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

  // Load project + org role
  useEffect(() => {
    if (status !== "authenticated" || !activeSlug || !id) return;

    const load = async () => {
      setLoading(true);
      setLoadError(null);
      setNotFound(false);
      try {
        const [proj, ws] = await Promise.all([
          getProject(activeSlug, id),
          getCurrentWorkspace(activeSlug),
        ]);
        setProject(proj);
        setCanMutate(ws.org_role_code === "owner" || ws.org_role_code === "admin");
      } catch (err) {
        if (err instanceof ApiError) {
          if (
            err.code === "project.not_found" ||
            err.code === "validation.invalid_input" // bad uuid → treat as not-found
          ) {
            setNotFound(true);
          } else if (
            err.code === "tenant.workspace_not_found" ||
            err.code === "tenant.forbidden"
          ) {
            clearActiveSlug();
            router.push("/workspaces");
          } else {
            setLoadError(errorMessage(err.code, lang));
          }
        } else {
          setLoadError(errorMessage("UNKNOWN_ERROR", lang));
        }
      } finally {
        setLoading(false);
      }
    };

    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [status, activeSlug, id, lang]);

  function startEdit() {
    if (!project) return;
    setName(project.project_name);
    setSlug(project.slug ?? "");
    setProjectType(project.project_type_code);
    setRequestingUnit(project.requesting_unit ?? "");
    setDescription(project.description ?? "");
    setStartDate(project.start_date ?? "");
    setEndDate(project.end_date ?? "");
    setNameError(null);
    setSlugError(null);
    setRequestingUnitError(null);
    setDescriptionError(null);
    setStartDateError(null);
    setEndDateError(null);
    setTypeError(null);
    setFormError(null);
    setIsEditing(true);
  }

  function cancelEdit() {
    setIsEditing(false);
  }

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    if (!activeSlug || !project) return;
    setNameError(null);
    setSlugError(null);
    setRequestingUnitError(null);
    setDescriptionError(null);
    setStartDateError(null);
    setEndDateError(null);
    setTypeError(null);
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

    let rangeErr: string | null = null;
    if (!startErr && !endErr) {
      rangeErr = validateDateRange(startDate, endDate);
      if (rangeErr) setEndDateError(t(rangeErr, lang));
    }

    if (nameErr || slugErr || reqUnitErr || descErr || startErr || endErr || rangeErr) return;

    // PUT is a full replace: omitting an optional clears it to NULL. Trim-and-omit.
    const input: UpdateProjectInput = {
      project_name: name.trim(),
      project_type_code: projectType,
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

    setSaving(true);
    try {
      const updated = await updateProject(activeSlug, project.id, input);
      setProject(updated);
      setIsEditing(false);
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
          // On edit, only project_type_code is possible.
          const f = (err.details as unknown as { field?: string } | undefined)?.field;
          if (f === "project_type_code") setTypeError(errorMessage(err.code, lang));
          setFormError(errorMessage("project.invalid_master_code", lang));
        } else if (err.code === "project.not_found") {
          setNotFound(true);
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
      setSaving(false);
    }
  }

  function onStatusSelect(value: string) {
    if (!project || value === project.project_status_code) return;
    setPendingStatus(value);
    setStatusDialogOpen(true);
  }

  async function confirmStatusChange() {
    if (!activeSlug || !project || !pendingStatus) return;
    setActionError(null);
    setStatusSaving(true);
    try {
      const updated = await changeProjectStatus(activeSlug, project.id, pendingStatus);
      setProject(updated);
      setStatusDialogOpen(false);
      setPendingStatus("");
    } catch (err) {
      setStatusDialogOpen(false);
      setPendingStatus("");
      if (err instanceof ApiError) {
        if (err.code === "project.not_found") {
          setNotFound(true);
        } else {
          setActionError(errorMessage(err.code, lang));
        }
      } else {
        setActionError(errorMessage("UNKNOWN_ERROR", lang));
      }
    } finally {
      setStatusSaving(false);
    }
  }

  function cancelStatusChange() {
    setStatusDialogOpen(false);
    setPendingStatus("");
  }

  async function confirmDelete() {
    if (!activeSlug || !project) return;
    setActionError(null);
    setDeleting(true);
    try {
      await deleteProject(activeSlug, project.id);
      router.push("/projects");
    } catch (err) {
      setDeleteDialogOpen(false);
      if (err instanceof ApiError) {
        if (err.code === "project.not_found") {
          setNotFound(true);
        } else {
          setActionError(errorMessage(err.code, lang));
        }
      } else {
        setActionError(errorMessage("UNKNOWN_ERROR", lang));
      }
    } finally {
      setDeleting(false);
    }
  }

  const inputClass =
    "rounded-md border px-3 py-2 text-sm outline-none transition focus:ring-2 focus:ring-zinc-400 border-zinc-300 bg-white";

  if (status === "loading" || loading) {
    return (
      <div className="flex-1 flex items-center justify-center text-zinc-500 text-sm">
        {t("msg.loading", lang)}
      </div>
    );
  }

  if (status === "anonymous" || !activeSlug) {
    return null;
  }

  if (notFound) {
    return (
      <div className="flex-1 flex items-center justify-center p-8">
        <div className="flex flex-col gap-4 max-w-sm w-full text-center">
          <p className="text-sm text-zinc-600">{t("page.project_detail.not_found", lang)}</p>
          <Button variant="secondary" onClick={() => router.push("/projects")}>
            {t("btn.back_to_projects", lang)}
          </Button>
        </div>
      </div>
    );
  }

  if (loadError) {
    return (
      <div className="flex-1 flex items-center justify-center p-8">
        <div className="flex flex-col gap-4 max-w-sm w-full">
          <Alert variant="error">{loadError}</Alert>
          <Button variant="secondary" onClick={() => router.push("/projects")}>
            {t("btn.back_to_projects", lang)}
          </Button>
        </div>
      </div>
    );
  }

  if (!project) return null;

  const targetStatusLabel = pendingStatus ? t(`status.${pendingStatus}`, lang) : "";

  return (
    <div className="flex-1 p-8 max-w-2xl mx-auto w-full flex flex-col gap-8">
      <Button variant="ghost" onClick={() => router.push("/projects")} className="self-start">
        {t("btn.back_to_projects", lang)}
      </Button>

      {actionError && <Alert variant="error">{actionError}</Alert>}

      {!isEditing ? (
        /* READ mode */
        <div className="bg-white rounded-2xl shadow-sm border border-zinc-200 p-6 flex flex-col gap-4">
          <div className="flex items-start justify-between gap-4">
            <div className="flex flex-col gap-2">
              <h1 className="text-2xl font-semibold text-zinc-800">{project.project_name}</h1>
              <div className="flex gap-2">
                <span className="inline-flex items-center rounded-full bg-zinc-100 px-2.5 py-0.5 text-xs font-medium text-zinc-700">
                  {t(`status.${project.project_status_code}`, lang)}
                </span>
                <span className="inline-flex items-center rounded-full bg-zinc-100 px-2.5 py-0.5 text-xs font-medium text-zinc-700">
                  {t(`type.${project.project_type_code}`, lang)}
                </span>
              </div>
            </div>
          </div>

          <dl className="flex flex-col gap-3 text-sm">
            <Field label={t("label.project_slug", lang)} value={project.slug ?? "—"} />
            <Field label={t("label.requesting_unit", lang)} value={project.requesting_unit ?? "—"} />
            <Field label={t("label.description", lang)} value={project.description ?? "—"} />
            <Field label={t("label.start_date", lang)} value={project.start_date ?? "—"} />
            <Field label={t("label.end_date", lang)} value={project.end_date ?? "—"} />
            <div className="flex flex-col gap-0.5">
              <dt className="text-xs font-medium text-zinc-500">{t("label.owner", lang)}</dt>
              <dd className="text-xs font-mono text-zinc-500">
                {project.owner_project_member_id ?? "—"}
              </dd>
            </div>
            <Field label={t("label.created_at", lang)} value={new Date(project.created_at).toLocaleString()} />
            <Field label={t("label.updated_at", lang)} value={new Date(project.updated_at).toLocaleString()} />
          </dl>

          {canMutate && (
            <div className="flex flex-wrap items-end gap-3 pt-2 border-t border-zinc-100">
              <Button variant="secondary" onClick={startEdit}>
                {t("btn.edit", lang)}
              </Button>
              <Select
                id="status-select"
                label={t("btn.change_status", lang)}
                value={project.project_status_code}
                options={statusOptions(lang)}
                onChange={(e) => onStatusSelect(e.target.value)}
                disabled={statusSaving}
              />
              <Button variant="danger" onClick={() => setDeleteDialogOpen(true)}>
                {t("btn.delete", lang)}
              </Button>
            </div>
          )}
        </div>
      ) : (
        /* EDIT mode — NO status select (status is PUT-excluded) */
        <div className="bg-white rounded-2xl shadow-sm border border-zinc-200 p-6 flex flex-col gap-4">
          <h2 className="text-lg font-semibold text-zinc-800">{t("btn.edit", lang)}</h2>

          {formError && <Alert variant="error">{formError}</Alert>}

          <form onSubmit={handleSave} className="flex flex-col gap-4" noValidate>
            <Input
              id="edit-name"
              label={t("label.project_name", lang)}
              value={name}
              onChange={(e) => setName(e.target.value)}
              error={nameError ?? undefined}
              disabled={saving}
              autoComplete="off"
            />
            <Input
              id="edit-slug"
              label={t("label.project_slug", lang)}
              value={slug}
              onChange={(e) => setSlug(e.target.value)}
              error={slugError ?? undefined}
              disabled={saving}
              autoComplete="off"
              placeholder="my-project"
            />
            <Select
              id="edit-type"
              label={t("label.project_type", lang)}
              value={projectType}
              options={typeOptions(lang)}
              onChange={(e) => setProjectType(e.target.value)}
              error={typeError ?? undefined}
              disabled={saving}
            />
            {/* status intentionally omitted — PUT does not accept project_status_code */}
            <Input
              id="edit-requesting-unit"
              label={t("label.requesting_unit", lang)}
              value={requestingUnit}
              onChange={(e) => setRequestingUnit(e.target.value)}
              error={requestingUnitError ?? undefined}
              disabled={saving}
              autoComplete="off"
            />
            <div className="flex flex-col gap-1">
              <label htmlFor="edit-description" className="text-sm font-medium text-zinc-700">
                {t("label.description", lang)}
              </label>
              <textarea
                id="edit-description"
                className={inputClass}
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                disabled={saving}
                rows={4}
              />
              {descriptionError && <p className="text-xs text-red-600">{descriptionError}</p>}
            </div>
            <Input
              id="edit-start-date"
              type="date"
              label={t("label.start_date", lang)}
              value={startDate}
              onChange={(e) => setStartDate(e.target.value)}
              error={startDateError ?? undefined}
              disabled={saving}
            />
            <Input
              id="edit-end-date"
              type="date"
              label={t("label.end_date", lang)}
              value={endDate}
              onChange={(e) => setEndDate(e.target.value)}
              error={endDateError ?? undefined}
              disabled={saving}
            />
            <div className="flex gap-2">
              <Button type="submit" loading={saving}>
                {t("btn.save", lang)}
              </Button>
              <Button type="button" variant="ghost" onClick={cancelEdit} disabled={saving}>
                {t("btn.cancel", lang)}
              </Button>
            </div>
          </form>
        </div>
      )}

      <ConfirmDialog
        open={statusDialogOpen}
        title={t("dialog.change_status.title", lang)}
        body={targetStatusLabel}
        confirmLabel={t("btn.change_status", lang)}
        cancelLabel={t("btn.cancel", lang)}
        loading={statusSaving}
        onConfirm={confirmStatusChange}
        onCancel={cancelStatusChange}
      />

      <ConfirmDialog
        open={deleteDialogOpen}
        title={t("dialog.delete.title", lang)}
        body={t("dialog.delete.body", lang)}
        confirmLabel={t("btn.delete", lang)}
        cancelLabel={t("btn.cancel", lang)}
        confirmVariant="danger"
        loading={deleting}
        onConfirm={confirmDelete}
        onCancel={() => setDeleteDialogOpen(false)}
      />
    </div>
  );
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col gap-0.5">
      <dt className="text-xs font-medium text-zinc-500">{label}</dt>
      <dd className="text-zinc-800 whitespace-pre-wrap break-words">{value}</dd>
    </div>
  );
}
