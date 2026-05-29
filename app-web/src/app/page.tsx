"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth-context";
import { useWorkspace } from "@/lib/workspace-context";
import { useLang } from "@/lib/lang-context";
import { t, errorMessage } from "@/lib/i18n";
import { validateEmail } from "@/lib/validation";
import { getCurrentWorkspace, inviteMember } from "@/lib/workspace-api";
import { ApiError } from "@/lib/api";
import type { CurrentWorkspace } from "@/lib/types";
import { Input, Button, Alert, Select } from "@/components/ui";

const INVITE_ROLES = ["admin", "executive", "user"] as const;
type InviteRole = (typeof INVITE_ROLES)[number];

export default function Home() {
  const { status, logout } = useAuth();
  const { activeSlug, clearActiveSlug } = useWorkspace();
  const { lang } = useLang();
  const router = useRouter();

  // Workspace home state
  const [workspace, setWorkspace] = useState<CurrentWorkspace | null>(null);
  const [wsLoading, setWsLoading] = useState(false);
  const [wsError, setWsError] = useState<string | null>(null);

  // Invite form state
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

  // Load current workspace
  useEffect(() => {
    if (status !== "authenticated" || !activeSlug) return;

    const load = async () => {
      setWsLoading(true);
      setWsError(null);
      try {
        const ws = await getCurrentWorkspace(activeSlug);
        setWorkspace(ws);
      } catch (err) {
        if (
          err instanceof ApiError &&
          (err.code === "tenant.workspace_not_found" || err.code === "tenant.forbidden")
        ) {
          // Stale/removed active slug — clear and redirect
          clearActiveSlug();
          router.push("/workspaces");
        } else {
          setWsError(
            err instanceof ApiError
              ? errorMessage(err.code, lang)
              : errorMessage("UNKNOWN_ERROR", lang)
          );
        }
      } finally {
        setWsLoading(false);
      }
    };

    void load();
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [status, activeSlug]);

  async function handleLogout() {
    await logout();
    clearActiveSlug();
    router.push("/login");
  }

  function handleSwitchWorkspace() {
    clearActiveSlug();
    router.push("/workspaces");
  }

  async function handleInvite(e: React.FormEvent) {
    e.preventDefault();
    setInviteEmailError(null);
    setInviteFormError(null);
    setInviteSuccess(false);

    const emailErr = validateEmail(inviteEmail);
    if (emailErr) {
      setInviteEmailError(t(emailErr, lang));
      return;
    }

    if (!activeSlug) return;

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

  if (status === "loading") {
    return (
      <div className="flex-1 flex items-center justify-center text-zinc-500 text-sm">
        {t("msg.loading", lang)}
      </div>
    );
  }

  if (status === "anonymous" || !activeSlug) {
    // Redirect in progress
    return null;
  }

  if (wsLoading) {
    return (
      <div className="flex-1 flex items-center justify-center text-zinc-500 text-sm">
        {t("msg.loading", lang)}
      </div>
    );
  }

  if (wsError) {
    return (
      <div className="flex-1 flex items-center justify-center p-8">
        <div className="flex flex-col gap-4 max-w-sm w-full">
          <Alert variant="error">{wsError}</Alert>
          <Button variant="secondary" onClick={handleSwitchWorkspace}>
            {t("btn.switch_workspace", lang)}
          </Button>
        </div>
      </div>
    );
  }

  const canInvite =
    workspace?.org_role_code === "owner" || workspace?.org_role_code === "admin";

  const roleOptions = INVITE_ROLES.map((r) => ({
    value: r,
    label: t(`role.${r}`, lang),
  }));

  return (
    <div className="flex-1 p-8 max-w-2xl mx-auto w-full flex flex-col gap-8">
      {/* Workspace header */}
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold text-zinc-800">
            {workspace?.workspace_name ?? activeSlug}
          </h1>
          {workspace && (
            <p className="text-sm text-zinc-500 mt-1">
              {t("label.role", lang)}: {workspace.org_role_code}
            </p>
          )}
        </div>
        <div className="flex gap-2 flex-shrink-0">
          <Button variant="ghost" onClick={handleSwitchWorkspace}>
            {t("btn.switch_workspace", lang)}
          </Button>
          <Button variant="secondary" onClick={handleLogout}>
            {t("btn.logout", lang)}
          </Button>
        </div>
      </div>

      {/* Invite member form — only for owner/admin */}
      {canInvite && (
        <div className="bg-white rounded-2xl shadow-sm border border-zinc-200 p-6 flex flex-col gap-4">
          <h2 className="text-lg font-semibold text-zinc-800">
            {t("page.home.invite_heading", lang)}
          </h2>

          {inviteFormError && <Alert variant="error">{inviteFormError}</Alert>}
          {inviteSuccess && (
            <Alert variant="success">{t("msg.invitation_sent", lang)}</Alert>
          )}

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
              label={t("label.invite_role", lang)}
              value={inviteRole}
              onChange={(e) => setInviteRole(e.target.value as InviteRole)}
              options={roleOptions}
              disabled={inviting}
            />
            <Button type="submit" loading={inviting}>
              {t("btn.send_invite", lang)}
            </Button>
          </form>
        </div>
      )}
    </div>
  );
}
