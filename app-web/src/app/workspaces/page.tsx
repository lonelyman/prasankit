"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth-context";
import { useWorkspace } from "@/lib/workspace-context";
import { useLang } from "@/lib/lang-context";
import { t, errorMessage } from "@/lib/i18n";
import { validateEmail, validateWorkspaceName, validateSlug } from "@/lib/validation";
import { listWorkspaces, createWorkspace } from "@/lib/workspace-api";
import { ApiError } from "@/lib/api";
import type { WorkspaceWithRole } from "@/lib/types";
import { Input, Button, Alert } from "@/components/ui";

export default function WorkspacesPage() {
  const { status, account } = useAuth();
  const { activeSlug, setActiveSlug, refreshWorkspaces } = useWorkspace();
  const { lang } = useLang();
  const router = useRouter();

  // Workspace list state
  const [workspaces, setWorkspaces] = useState<WorkspaceWithRole[]>([]);
  const [listLoading, setListLoading] = useState(true);
  const [listError, setListError] = useState<string | null>(null);

  // Create form state
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [contactEmail, setContactEmail] = useState("");
  const [nameError, setNameError] = useState<string | null>(null);
  const [slugError, setSlugError] = useState<string | null>(null);
  const [contactEmailError, setContactEmailError] = useState<string | null>(null);
  const [formError, setFormError] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [showCreate, setShowCreate] = useState(false); // when you already have workspaces, create is a collapsed secondary action

  // Redirect anonymous users
  useEffect(() => {
    if (status === "anonymous") {
      router.push("/login");
    }
  }, [status, router]);

  // Pre-fill the workspace contact email with the signed-in user's primary email
  // (a sensible default — editable; the workspace contact email is an org-level field
  // that may differ from the creator's login email). One-shot: won't clobber edits.
  const contactPrefilledRef = useRef(false);
  useEffect(() => {
    if (!contactPrefilledRef.current && account?.primary_email) {
      setContactEmail(account.primary_email);
      contactPrefilledRef.current = true;
    }
  }, [account]);

  // Load workspace list
  useEffect(() => {
    if (status !== "authenticated") return;

    const load = async () => {
      setListLoading(true);
      try {
        const items = await listWorkspaces();
        setWorkspaces(items);
      } catch (err) {
        setListError(
          err instanceof ApiError ? errorMessage(err.code, lang) : errorMessage("UNKNOWN_ERROR", lang)
        );
      } finally {
        setListLoading(false);
      }
    };

    void load();
  }, [status, lang]);

  // Auto-enter when there is exactly ONE workspace and none is active yet (login landing):
  // skip the picker so login → work is one click. Does NOT fire when activeSlug is set
  // (e.g. arriving via the header workspace link), so switching / creating a 2nd workspace
  // stays reachable from /workspaces.
  useEffect(() => {
    if (!listLoading && !listError && !activeSlug && workspaces.length === 1) {
      setActiveSlug(workspaces[0].slug);
      router.replace("/projects");
    }
  }, [listLoading, listError, activeSlug, workspaces, setActiveSlug, router]);

  if (status === "loading") {
    return (
      <div className="flex-1 flex items-center justify-center text-slate-500 text-sm">
        {t("msg.loading", lang)}
      </div>
    );
  }

  if (status === "anonymous") {
    return null;
  }

  function handleEnter(ws: WorkspaceWithRole) {
    setActiveSlug(ws.slug);
    router.push("/");
  }

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setNameError(null);
    setSlugError(null);
    setContactEmailError(null);
    setFormError(null);

    const nameErr = validateWorkspaceName(name);
    const slugErr = validateSlug(slug);
    const emailErr = validateEmail(contactEmail);

    if (nameErr) setNameError(t(nameErr, lang));
    if (slugErr) setSlugError(t(slugErr, lang));
    if (emailErr) setContactEmailError(t(emailErr, lang));

    if (nameErr || slugErr || emailErr) return;

    setCreating(true);
    try {
      const created = await createWorkspace({
        name: name.trim(),
        slug,
        contact_email: contactEmail.trim(),
      });
      // Refresh the cached list so the header switcher shows the new workspace.
      void refreshWorkspaces();
      setActiveSlug(created.slug);
      router.push("/");
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.code === "validation.invalid_input" && err.details) {
          for (const detail of err.details) {
            if (detail.field === "name") setNameError(detail.message);
            else if (detail.field === "slug") setSlugError(detail.message);
            else if (detail.field === "contact_email") setContactEmailError(detail.message);
          }
        } else if (
          err.code === "workspace.slug_reserved" ||
          err.code === "workspace.slug_taken"
        ) {
          setSlugError(errorMessage(err.code, lang));
        } else {
          setFormError(errorMessage(err.code, lang));
        }
      } else {
        setFormError(errorMessage("UNKNOWN_ERROR", lang));
      }
    } finally {
      setCreating(false);
    }
  }

  const isEmpty = !listLoading && workspaces.length === 0;

  return (
    <div className="flex-1 p-8 max-w-2xl mx-auto w-full flex flex-col gap-8">
      {/* Workspace list or empty state heading */}
      <div>
        <h1 className="text-2xl font-semibold text-slate-800 mb-4">
          {isEmpty
            ? t("page.workspaces.empty_heading", lang)
            : t("page.workspaces.title", lang)}
        </h1>

        {listLoading && (
          <p className="text-sm text-slate-500">{t("msg.loading", lang)}</p>
        )}

        {listError && <Alert variant="error">{listError}</Alert>}

        {!listLoading && workspaces.length > 0 && (
          <ul className="flex flex-col gap-2">
            {workspaces.map((ws) => (
              <li
                key={ws.id}
                className="flex items-center justify-between rounded-lg border border-slate-200 bg-white px-4 py-3"
              >
                <div>
                  <p className="font-medium text-slate-800">{ws.workspace_name}</p>
                  <p className="text-xs text-slate-500">
                    {ws.slug} &middot; {ws.org_role_code}
                  </p>
                </div>
                <Button
                  variant="secondary"
                  onClick={() => handleEnter(ws)}
                >
                  {t("btn.enter_workspace", lang)}
                </Button>
              </li>
            ))}
          </ul>
        )}

        {isEmpty && !showCreate && (
          <div className="flex flex-col items-center gap-3 rounded-2xl border border-dashed border-slate-200 bg-white px-6 py-12 text-center">
            <p className="text-sm text-slate-500">{t("msg.invite_hint", lang)}</p>
            <Button onClick={() => setShowCreate(true)}>
              {t("btn.create_first_workspace", lang)}
            </Button>
          </div>
        )}
      </div>

      {/* Create workspace: a hero form when you have none (after the CTA),
          a collapsed secondary action when you already have workspaces (most
          people keep just one, so the form should not dominate the list every
          visit). */}
      {!showCreate ? (
        !isEmpty && (
          <div className="self-start">
            <Button variant="secondary" onClick={() => setShowCreate(true)}>
              {t("btn.create_new_workspace", lang)}
            </Button>
          </div>
        )
      ) : (
        <div className="bg-white rounded-2xl shadow-sm border border-slate-200 p-6 flex flex-col gap-4">
          <div className="flex items-center justify-between gap-4">
            <h2 className="text-lg font-semibold text-slate-800">
              {t("page.workspaces.create_heading", lang)}
            </h2>
            {!isEmpty && (
              <Button variant="ghost" onClick={() => setShowCreate(false)}>
                {t("btn.cancel", lang)}
              </Button>
            )}
          </div>

          {formError && <Alert variant="error">{formError}</Alert>}

          <form onSubmit={handleCreate} className="flex flex-col gap-4" noValidate>
          <Input
            id="workspace-name"
            label={t("label.workspace_name", lang)}
            value={name}
            onChange={(e) => setName(e.target.value)}
            error={nameError ?? undefined}
            disabled={creating}
            autoComplete="off"
          />
          <Input
            id="workspace-slug"
            label={t("label.slug", lang)}
            value={slug}
            onChange={(e) => setSlug(e.target.value)}
            error={slugError ?? undefined}
            disabled={creating}
            autoComplete="off"
            placeholder="my-workspace"
          />
          <Input
            id="workspace-contact-email"
            type="email"
            label={t("label.contact_email", lang)}
            value={contactEmail}
            onChange={(e) => setContactEmail(e.target.value)}
            error={contactEmailError ?? undefined}
            disabled={creating}
            autoComplete="email"
          />
          <Button type="submit" loading={creating}>
            {t("btn.create_workspace", lang)}
          </Button>
        </form>
        </div>
      )}
    </div>
  );
}
