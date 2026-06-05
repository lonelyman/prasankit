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
import { getCurrentWorkspace, inviteMember, listWorkspaceMembers } from "@/lib/workspace-api";
import {
  getProject,
  updateProject,
  deleteProject,
  changeProjectStatus,
} from "@/lib/project-api";
import type { UpdateProjectInput } from "@/lib/project-api";
import {
  listMembers,
  addMember,
  changeMemberRole,
  removeMember,
} from "@/lib/member-api";
import { validateEmail } from "@/lib/validation";
import { ApiError } from "@/lib/api";
import type { Project, ProjectMember, WorkspaceMember } from "@/lib/types";
import { statusOptions, typeOptions, statusColor } from "@/lib/project-masters";
import {
  projectRoleOptions,
  projectRoleLabel,
  projectRoleColor,
} from "@/lib/project-roles";
import { Input, Button, Alert, Select, Modal, ConfirmDialog, Badge, Breadcrumb } from "@/components/ui";

// Org roles invitable to a workspace (mirror of the recovered home invite form).
const INVITE_ROLES = ["admin", "executive", "user"] as const;
type InviteRole = (typeof INVITE_ROLES)[number];

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

  // Team / members state
  const [members, setMembers] = useState<ProjectMember[]>([]);
  const [membersLoading, setMembersLoading] = useState(true);
  const [membersError, setMembersError] = useState<string | null>(null);
  const [roleSavingId, setRoleSavingId] = useState<string | null>(null);
  const [removeDialogOpen, setRemoveDialogOpen] = useState(false);
  const [pendingMember, setPendingMember] = useState<ProjectMember | null>(null);
  const [removing, setRemoving] = useState(false);

  // Add-member modal state
  const [addOpen, setAddOpen] = useState(false);
  const [wsMembers, setWsMembers] = useState<WorkspaceMember[]>([]);
  const [wsMembersLoading, setWsMembersLoading] = useState(false);
  const [addMembershipId, setAddMembershipId] = useState("");
  const [addRole, setAddRole] = useState("member");
  const [addError, setAddError] = useState<string | null>(null);
  const [addMembershipError, setAddMembershipError] = useState<string | null>(null);
  const [addRoleError, setAddRoleError] = useState<string | null>(null);
  const [adding, setAdding] = useState(false);

  // Invite modal state (workspace-level invite, reinstated in the team area)
  const [inviteOpen, setInviteOpen] = useState(false);
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState<InviteRole>("user");
  const [inviteEmailError, setInviteEmailError] = useState<string | null>(null);
  const [inviteFormError, setInviteFormError] = useState<string | null>(null);
  const [inviteSuccess, setInviteSuccess] = useState(false);
  const [inviting, setInviting] = useState(false);

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

  // Load members once the project itself has loaded. Shared by the post-mutate
  // refetch (add/change/remove) so the list shows the loading row, never a stale
  // list. tenant.* → kick to /workspaces (mirrors the project-load branch); a
  // 403 here would be tenant.permission_denied and is surfaced inline, not here.
  async function loadMembers() {
    if (!activeSlug || !id) return;
    setMembersLoading(true);
    setMembersError(null);
    try {
      const result = await listMembers(activeSlug, id);
      setMembers(result.items);
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.code === "project.not_found") {
          setNotFound(true);
        } else if (
          err.code === "tenant.workspace_not_found" ||
          err.code === "tenant.forbidden"
        ) {
          clearActiveSlug();
          router.push("/workspaces");
        } else {
          setMembersError(errorMessage(err.code, lang));
        }
      } else {
        setMembersError(errorMessage("UNKNOWN_ERROR", lang));
      }
    } finally {
      setMembersLoading(false);
    }
  }

  // Fetch members after the project loads (project drives notFound/loadError).
  // loadMembers is reused by post-mutate refetches; its synchronous setLoading is
  // the intended "show the loading row" behavior, so the cascading-render lint is
  // suppressed here exactly as exhaustive-deps is for the sibling load effect.
  useEffect(() => {
    if (status !== "authenticated" || !activeSlug || !id || !project) return;
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void loadMembers();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [status, activeSlug, id, project, lang]);

  const ownerMemberId = project?.owner_project_member_id ?? null;
  const ownerName = ownerMemberId
    ? members.find((m) => m.id === ownerMemberId)?.display_name ?? null
    : null;

  function openAddModal() {
    setAddMembershipId("");
    setAddRole("member");
    setAddError(null);
    setAddMembershipError(null);
    setAddRoleError(null);
    setAddOpen(true);
    void loadWorkspaceMembers();
  }

  async function loadWorkspaceMembers() {
    if (!activeSlug) return;
    setWsMembersLoading(true);
    try {
      const result = await listWorkspaceMembers(activeSlug);
      setWsMembers(result.items);
    } catch (err) {
      if (err instanceof ApiError) {
        setAddError(errorMessage(err.code, lang));
      } else {
        setAddError(errorMessage("UNKNOWN_ERROR", lang));
      }
    } finally {
      setWsMembersLoading(false);
    }
  }

  // Eligible = active ws members not already on the project. The project list is
  // active-only (BE filters removed_at IS NULL), so no removed-at clause needed;
  // a re-added person reappears once they leave the active list.
  const eligibleMembers = wsMembers.filter(
    (ws) => !members.some((m) => m.workspace_membership_id === ws.workspace_membership_id)
  );

  async function handleAddMember(e: React.FormEvent) {
    e.preventDefault();
    if (!activeSlug || !project) return;
    setAddError(null);
    setAddMembershipError(null);
    setAddRoleError(null);

    if (!addMembershipId) {
      setAddMembershipError(t("label.select_member", lang));
      return;
    }

    setAdding(true);
    try {
      await addMember(activeSlug, project.id, {
        workspace_membership_id: addMembershipId,
        project_role_code: addRole,
      });
      setAddOpen(false);
      await loadMembers();
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.code === "project_member.membership_not_eligible") {
          setAddMembershipError(errorMessage(err.code, lang));
        } else if (err.code === "project_member.invalid_role_code") {
          setAddRoleError(errorMessage(err.code, lang));
        } else if (err.code === "project_member.already_member") {
          setAddError(errorMessage(err.code, lang));
        } else if (err.code === "project.not_found") {
          setAddOpen(false);
          setNotFound(true);
        } else if (
          err.code === "tenant.workspace_not_found" ||
          err.code === "tenant.forbidden"
        ) {
          clearActiveSlug();
          router.push("/workspaces");
        } else {
          setAddError(errorMessage(err.code ?? "UNKNOWN_ERROR", lang));
        }
      } else {
        setAddError(errorMessage("UNKNOWN_ERROR", lang));
      }
    } finally {
      setAdding(false);
    }
  }

  async function onRoleSelect(member: ProjectMember, newCode: string) {
    if (!activeSlug || !project) return;
    if (newCode === member.project_role_code) return; // no-op: don't call
    setActionError(null);
    setRoleSavingId(member.id);
    try {
      await changeMemberRole(activeSlug, project.id, member.id, newCode);
      await loadMembers();
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.code === "project_member.owner_immutable") {
          // Stale-client reconcile: surface + refetch to re-sync the owner row.
          setActionError(errorMessage(err.code, lang));
          await loadMembers();
        } else if (
          err.code === "project_member.not_found" ||
          err.code === "project.not_found"
        ) {
          await loadMembers();
        } else if (
          err.code === "tenant.workspace_not_found" ||
          err.code === "tenant.forbidden"
        ) {
          clearActiveSlug();
          router.push("/workspaces");
        } else {
          // incl. project_member.invalid_role_code, tenant.permission_denied (inline, no redirect)
          setActionError(errorMessage(err.code, lang));
        }
      } else {
        setActionError(errorMessage("UNKNOWN_ERROR", lang));
      }
    } finally {
      setRoleSavingId(null);
    }
  }

  function openRemoveDialog(member: ProjectMember) {
    setPendingMember(member);
    setRemoveDialogOpen(true);
  }

  async function confirmRemoveMember() {
    if (!activeSlug || !project || !pendingMember) return;
    setActionError(null);
    setRemoving(true);
    try {
      await removeMember(activeSlug, project.id, pendingMember.id);
      setRemoveDialogOpen(false);
      setPendingMember(null);
      await loadMembers();
    } catch (err) {
      setRemoveDialogOpen(false);
      setPendingMember(null);
      if (err instanceof ApiError) {
        if (err.code === "project_member.owner_immutable") {
          setActionError(errorMessage(err.code, lang));
          await loadMembers();
        } else if (
          err.code === "project_member.not_found" ||
          err.code === "project.not_found"
        ) {
          await loadMembers();
        } else if (
          err.code === "tenant.workspace_not_found" ||
          err.code === "tenant.forbidden"
        ) {
          clearActiveSlug();
          router.push("/workspaces");
        } else {
          setActionError(errorMessage(err.code, lang));
        }
      } else {
        setActionError(errorMessage("UNKNOWN_ERROR", lang));
      }
    } finally {
      setRemoving(false);
    }
  }

  function openInviteModal() {
    setInviteEmail("");
    setInviteRole("user");
    setInviteEmailError(null);
    setInviteFormError(null);
    setInviteSuccess(false);
    setInviteOpen(true);
  }

  async function handleInvite(e: React.FormEvent) {
    e.preventDefault();
    if (!activeSlug) return;
    setInviteEmailError(null);
    setInviteFormError(null);
    setInviteSuccess(false);

    const emailErr = validateEmail(inviteEmail);
    if (emailErr) {
      setInviteEmailError(t(emailErr, lang));
      return;
    }

    setInviting(true);
    try {
      await inviteMember(activeSlug, {
        email: inviteEmail.trim(),
        org_role_code: inviteRole,
      });
      setInviteSuccess(true);
      setInviteEmail("");
      setInviteRole("user");
    } catch (err) {
      if (err instanceof ApiError) {
        setInviteFormError(errorMessage(err.code, lang));
      } else {
        setInviteFormError(errorMessage("UNKNOWN_ERROR", lang));
      }
    } finally {
      setInviting(false);
    }
  }

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
    "rounded-md border px-3 py-2 text-sm outline-none transition focus:ring-2 focus:ring-slate-400 border-slate-300 bg-white";

  if (status === "loading" || loading) {
    return (
      <div className="flex-1 flex items-center justify-center text-slate-500 text-sm">
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
          <p className="text-sm text-slate-600">{t("page.project_detail.not_found", lang)}</p>
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
      <Breadcrumb
        items={[
          { label: t("nav.projects", lang), href: "/projects" },
          { label: project.project_name },
        ]}
      />

      {actionError && <Alert variant="error">{actionError}</Alert>}

      {!isEditing ? (
        /* READ mode */
        <div className="bg-white rounded-2xl shadow-sm border border-slate-200 p-6 flex flex-col gap-4">
          <div className="flex items-start justify-between gap-4">
            <div className="flex flex-col gap-2">
              <h1 className="text-2xl font-semibold text-slate-800">{project.project_name}</h1>
              <div className="flex gap-2">
                <Badge color={statusColor(project.project_status_code)}>
                  {t(`status.${project.project_status_code}`, lang)}
                </Badge>
                <Badge>{t(`type.${project.project_type_code}`, lang)}</Badge>
              </div>
            </div>
          </div>

          <dl className="flex flex-col gap-3 text-sm">
            <Field label={t("label.project_slug", lang)} value={project.slug ?? "—"} />
            <Field label={t("label.requesting_unit", lang)} value={project.requesting_unit ?? "—"} />
            <Field label={t("label.description", lang)} value={project.description ?? "—"} />
            <Field label={t("label.start_date", lang)} value={project.start_date ?? "—"} />
            <Field label={t("label.end_date", lang)} value={project.end_date ?? "—"} />
            {/* Owner resolved to display_name from the members list (loading-guarded
                so it never flashes the raw UUID). The Team section below surfaces
                the full owner row; this keeps the Owner field on the project card. */}
            <Field
              label={t("label.owner", lang)}
              value={
                ownerMemberId
                  ? membersLoading
                    ? t("msg.loading", lang)
                    : ownerName ?? "—"
                  : "—"
              }
            />
            <Field label={t("label.created_at", lang)} value={new Date(project.created_at).toLocaleString()} />
            <Field label={t("label.updated_at", lang)} value={new Date(project.updated_at).toLocaleString()} />
          </dl>

          {canMutate && (
            <div className="flex flex-wrap items-end gap-3 pt-2 border-t border-slate-100">
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
        <div className="bg-white rounded-2xl shadow-sm border border-slate-200 p-6 flex flex-col gap-4">
          <h2 className="text-lg font-semibold text-slate-800">{t("btn.edit", lang)}</h2>

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
              <label htmlFor="edit-description" className="text-sm font-medium text-slate-700">
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

      {/* Team section — visible to all org roles; write controls gated on canMutate. */}
      <div className="bg-white rounded-2xl shadow-sm border border-slate-200 p-6 flex flex-col gap-4">
        <div className="flex items-center justify-between gap-4">
          <h2 className="text-lg font-semibold text-slate-800">{t("team.heading", lang)}</h2>
          {canMutate && (
            <div className="flex items-center gap-2">
              <Button variant="ghost" onClick={openInviteModal}>
                {t("team.invite_member", lang)}
              </Button>
              <Button variant="secondary" onClick={openAddModal}>
                {t("team.add_member", lang)}
              </Button>
            </div>
          )}
        </div>

        {membersError && <Alert variant="error">{membersError}</Alert>}

        {membersLoading ? (
          <p className="text-sm text-slate-500">{t("msg.loading", lang)}</p>
        ) : members.length === 0 ? (
          // count===0 is the legacy / owner-less fallback (not the normal path).
          <p className="text-sm text-slate-500">{t("team.empty", lang)}</p>
        ) : (
          <ul className="flex flex-col divide-y divide-slate-100">
            {members.map((m) => {
              const isOwner = m.id === ownerMemberId;
              const rowSaving = roleSavingId === m.id;
              return (
                <li
                  key={m.id}
                  className="flex flex-wrap items-center justify-between gap-3 py-3 first:pt-0 last:pb-0"
                >
                  <div className="flex items-center gap-2 min-w-0">
                    <span className="text-sm text-slate-800 truncate">{m.display_name ?? "—"}</span>
                    <Badge color={projectRoleColor(m.project_role_code)}>
                      {projectRoleLabel(m.project_role_code, lang)}
                    </Badge>
                    {isOwner && (
                      <Badge color={projectRoleColor("project_owner")}>
                        {t("team.owner_badge", lang)}
                      </Badge>
                    )}
                  </div>

                  {canMutate &&
                    (isOwner ? (
                      <span className="text-xs text-slate-400 max-w-xs text-right">
                        {t("team.owner_locked_hint", lang)}
                      </span>
                    ) : (
                      <div className="flex items-center gap-2">
                        <Select
                          id={`role-${m.id}`}
                          options={projectRoleOptions(lang)}
                          value={m.project_role_code}
                          onChange={(e) => onRoleSelect(m, e.target.value)}
                          disabled={rowSaving}
                        />
                        <Button
                          variant="danger"
                          onClick={() => openRemoveDialog(m)}
                          disabled={rowSaving}
                        >
                          {t("btn.remove", lang)}
                        </Button>
                      </div>
                    ))}
                </li>
              );
            })}
          </ul>
        )}

        {/* Owner-only sparse state: a viewer/executive sees "no other members yet". */}
        {!membersLoading &&
          !membersError &&
          !canMutate &&
          members.length === 1 &&
          members[0].id === ownerMemberId && (
            <p className="text-sm text-slate-500">{t("team.empty", lang)}</p>
          )}
      </div>

      {/* Add-member modal */}
      <Modal open={addOpen} title={t("dialog.add_member.title", lang)} onClose={() => setAddOpen(false)}>
        {addError && <Alert variant="error">{addError}</Alert>}
        {wsMembersLoading ? (
          <p className="text-sm text-slate-500">{t("msg.loading", lang)}</p>
        ) : eligibleMembers.length === 0 ? (
          <div className="flex flex-col gap-3 text-sm text-slate-600">
            <p>{t("team.no_eligible_members", lang)}</p>
            <Button
              variant="ghost"
              className="self-start"
              onClick={() => {
                setAddOpen(false);
                openInviteModal();
              }}
            >
              {t("team.invite_member", lang)}
            </Button>
          </div>
        ) : (
          <form onSubmit={handleAddMember} className="flex flex-col gap-4" noValidate>
            <Select
              id="add-member-select"
              label={t("label.select_member", lang)}
              value={addMembershipId}
              options={[
                { value: "", label: t("label.select_member", lang) },
                ...eligibleMembers.map((ws) => ({
                  value: ws.workspace_membership_id,
                  label: ws.display_name,
                })),
              ]}
              onChange={(e) => setAddMembershipId(e.target.value)}
              error={addMembershipError ?? undefined}
              disabled={adding}
            />
            <Select
              id="add-member-role"
              label={t("label.member_role", lang)}
              value={addRole}
              options={projectRoleOptions(lang)}
              onChange={(e) => setAddRole(e.target.value)}
              error={addRoleError ?? undefined}
              disabled={adding}
            />
            <div className="flex justify-end gap-2">
              <Button
                type="button"
                variant="ghost"
                onClick={() => setAddOpen(false)}
                disabled={adding}
              >
                {t("btn.cancel", lang)}
              </Button>
              <Button type="submit" loading={adding}>
                {t("btn.add", lang)}
              </Button>
            </div>
          </form>
        )}
      </Modal>

      {/* Invite modal — workspace-level invite (owner/admin), reinstated in team area. */}
      <Modal open={inviteOpen} title={t("team.invite_member", lang)} onClose={() => setInviteOpen(false)}>
        {inviteFormError && <Alert variant="error">{inviteFormError}</Alert>}
        {inviteSuccess && <Alert variant="success">{t("msg.invitation_sent", lang)}</Alert>}
        <form onSubmit={handleInvite} className="flex flex-col gap-4" noValidate>
          <Input
            id="invite-email"
            type="email"
            label={t("label.invite_email", lang)}
            value={inviteEmail}
            onChange={(e) => setInviteEmail(e.target.value)}
            error={inviteEmailError ?? undefined}
            disabled={inviting}
            autoComplete="email"
          />
          <Select
            id="invite-role"
            label={t("label.role", lang)}
            value={inviteRole}
            options={INVITE_ROLES.map((r) => ({ value: r, label: t(`role.${r}`, lang) }))}
            onChange={(e) => setInviteRole(e.target.value as InviteRole)}
            disabled={inviting}
          />
          <div className="flex justify-end gap-2">
            <Button
              type="button"
              variant="ghost"
              onClick={() => setInviteOpen(false)}
              disabled={inviting}
            >
              {t("btn.cancel", lang)}
            </Button>
            <Button type="submit" loading={inviting}>
              {t("btn.send_invite", lang)}
            </Button>
          </div>
        </form>
      </Modal>

      <ConfirmDialog
        open={removeDialogOpen}
        title={t("dialog.remove_member.title", lang)}
        body={t("dialog.remove_member.body", lang)}
        confirmLabel={t("btn.remove", lang)}
        cancelLabel={t("btn.cancel", lang)}
        confirmVariant="danger"
        loading={removing}
        onConfirm={confirmRemoveMember}
        onCancel={() => {
          setRemoveDialogOpen(false);
          setPendingMember(null);
        }}
      />

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
      <dt className="text-xs font-medium text-slate-500">{label}</dt>
      <dd className="text-slate-800 whitespace-pre-wrap break-words">{value}</dd>
    </div>
  );
}
