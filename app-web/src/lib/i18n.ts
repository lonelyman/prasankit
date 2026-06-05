export type Lang = "th" | "en";
export const DEFAULT_LANG: Lang = "th";

// ---------------------------------------------------------------------------
// UI strings — labels, buttons, page copy, validation error messages
// ---------------------------------------------------------------------------

const UI_STRINGS: Record<string, Record<Lang, string>> = {
  // --- common ---
  "btn.submit": { th: "ส่ง", en: "Submit" },
  "btn.login": { th: "เข้าสู่ระบบ", en: "Log in" },
  "btn.logout": { th: "ออกจากระบบ", en: "Log out" },
  "btn.signup": { th: "สมัครสมาชิก", en: "Sign up" },
  "btn.resend": { th: "ส่งอีเมลยืนยันอีกครั้ง", en: "Resend verification email" },
  "btn.reset_request": { th: "ส่งลิงก์รีเซ็ตรหัสผ่าน", en: "Send reset link" },
  "btn.reset_confirm": { th: "รีเซ็ตรหัสผ่าน", en: "Reset password" },
  "label.email": { th: "อีเมล", en: "Email" },
  "label.password": { th: "รหัสผ่าน", en: "Password" },
  "label.display_name": { th: "ชื่อที่แสดง", en: "Display name" },
  "label.new_password": { th: "รหัสผ่านใหม่", en: "New password" },
  "label.confirm_password": { th: "ยืนยันรหัสผ่าน", en: "Confirm password" },

  // --- page titles / headings ---
  "page.login.title": { th: "เข้าสู่ระบบ", en: "Log in" },
  "page.signup.title": { th: "สมัครสมาชิก", en: "Sign up" },
  "page.verify_email.title": { th: "ยืนยันอีเมล", en: "Verify email" },
  "page.reset_password.title": { th: "รีเซ็ตรหัสผ่าน", en: "Reset password" },

  // --- navigation links ---
  "link.to_signup": { th: "ยังไม่มีบัญชี? สมัครสมาชิก", en: "No account? Sign up" },
  "link.to_login": { th: "มีบัญชีแล้ว? เข้าสู่ระบบ", en: "Have an account? Log in" },
  "link.forgot_password": { th: "ลืมรหัสผ่าน?", en: "Forgot password?" },

  // --- messages ---
  "msg.signup_success": {
    th: "สมัครสมาชิกสำเร็จ! กรุณาตรวจสอบอีเมลเพื่อยืนยันบัญชีก่อนเข้าสู่ระบบ",
    en: "Signed up successfully! Please check your email to verify your account before logging in.",
  },
  "msg.verify_success": {
    th: "ยืนยันอีเมลสำเร็จ! คุณสามารถเข้าสู่ระบบได้แล้ว",
    en: "Email verified successfully! You can now log in.",
  },
  "msg.verifying": { th: "กำลังยืนยันอีเมล…", en: "Verifying email…" },
  "msg.verify_missing_token": {
    th: "ไม่พบโทเค็นยืนยัน กรุณาคลิกลิงก์จากอีเมลหรือขอลิงก์ใหม่ด้านล่าง",
    en: "No verification token found. Please click the link in your email or request a new one below.",
  },
  "msg.resend_sent": {
    th: "หากบัญชีนี้มีอยู่ในระบบ เราได้ส่งลิงก์ยืนยันใหม่ไปยังอีเมลของคุณแล้ว",
    en: "If an account with that email exists, we've sent a new verification link.",
  },
  "msg.reset_request_sent": {
    th: "หากบัญชีนี้มีอยู่ในระบบ เราได้ส่งลิงก์รีเซ็ตรหัสผ่านไปยังอีเมลของคุณแล้ว",
    en: "If an account with that email exists, we've sent a password reset link.",
  },
  "msg.reset_success": {
    th: "รีเซ็ตรหัสผ่านสำเร็จ! คุณสามารถเข้าสู่ระบบด้วยรหัสผ่านใหม่ได้แล้ว",
    en: "Password reset successfully! You can now log in with your new password.",
  },
  "msg.passwords_no_match": {
    th: "รหัสผ่านไม่ตรงกัน",
    en: "Passwords do not match.",
  },
  "msg.email_not_verified_hint": {
    th: "บัญชีของคุณยังไม่ได้ยืนยันอีเมล กรุณาตรวจสอบกล่องจดหมายหรือขอลิงก์ใหม่",
    en: "Your account email has not been verified. Please check your inbox or request a new link.",
  },
  "msg.loading": { th: "กำลังโหลด…", en: "Loading…" },
  "msg.logged_in_as": { th: "เข้าสู่ระบบในฐานะ", en: "Logged in as" },
  "msg.resend_email_label": {
    th: "ขอลิงก์ยืนยันอีเมลใหม่",
    en: "Request a new verification link",
  },
  "msg.reset_request_label": {
    th: "กรอกอีเมลของคุณเพื่อรับลิงก์รีเซ็ตรหัสผ่าน",
    en: "Enter your email to receive a password reset link.",
  },

  // --- validation error keys (returned from validation.ts) ---
  "invalid_email": { th: "กรุณากรอกอีเมลที่ถูกต้อง", en: "Please enter a valid email address." },
  "password_too_short": { th: "รหัสผ่านต้องมีอย่างน้อย 8 ตัวอักษร", en: "Password must be at least 8 characters." },
  "password_mismatch": { th: "รหัสผ่านยืนยันไม่ตรงกัน", en: "Passwords do not match." },
  "display_name_required": { th: "กรุณากรอกชื่อที่แสดง", en: "Display name is required." },
  "display_name_too_long": { th: "ชื่อที่แสดงต้องไม่เกิน 100 ตัวอักษร", en: "Display name must be 100 characters or fewer." },
  "workspace_name_required": { th: "กรุณากรอกชื่อ workspace", en: "Workspace name is required." },
  "workspace_name_too_long": { th: "ชื่อ workspace ต้องไม่เกิน 100 ตัวอักษร", en: "Workspace name must be 100 characters or fewer." },
  "slug_invalid": {
    th: "Slug ต้องใช้ตัวอักษรพิมพ์เล็ก ตัวเลข หรือขีดกลาง และต้องเริ่มต้น/ลงท้ายด้วยตัวอักษรหรือตัวเลข",
    en: "Slug must use lowercase letters, numbers, or hyphens, and must start and end with a letter or number.",
  },

  // --- workspace list / empty-state ---
  "page.workspaces.title": { th: "Workspaces ของคุณ", en: "Your Workspaces" },
  "page.workspaces.empty_heading": { th: "ยังไม่มี Workspace", en: "No workspaces yet" },
  "msg.invite_hint": {
    th: "มีคำเชิญอยู่? ตรวจสอบอีเมลของคุณเพื่อคลิกลิงก์",
    en: "Have an invitation? Check your email for the link.",
  },
  "page.workspaces.create_heading": { th: "สร้าง Workspace ใหม่", en: "Create a new workspace" },

  // --- workspace form labels ---
  "label.workspace_name": { th: "ชื่อ Workspace", en: "Workspace name" },
  "label.slug": { th: "Slug (URL identifier)", en: "Slug (URL identifier)" },
  "label.contact_email": { th: "อีเมลติดต่อ", en: "Contact email" },

  // --- workspace actions ---
  "btn.enter_workspace": { th: "เข้าใช้งาน", en: "Enter" },
  "btn.create_workspace": { th: "สร้าง Workspace", en: "Create workspace" },
  "btn.create_new_workspace": { th: "+ สร้าง Workspace ใหม่", en: "+ Create new workspace" },
  "btn.switch_workspace": { th: "เปลี่ยน Workspace", en: "Switch workspace" },
  "btn.new_workspace": { th: "+ Workspace ใหม่", en: "+ New workspace" },
  "btn.create_first_workspace": { th: "สร้าง Workspace แรกของคุณ", en: "Create your first workspace" },
  "nav.switch_workspace": { th: "เปลี่ยน Workspace", en: "Switch workspace" },

  // --- workspace home ---
  "page.home.title": { th: "หน้าหลัก", en: "Home" },
  "label.role": { th: "บทบาท", en: "Role" },

  // --- invite form ---
  "label.invite_email": { th: "อีเมลที่ต้องการเชิญ", en: "Invite email" },
  "label.invite_role": { th: "บทบาท", en: "Role" },
  "btn.send_invite": { th: "ส่งคำเชิญ", en: "Send invite" },
  "msg.invitation_sent": { th: "ส่งคำเชิญสำเร็จแล้ว", en: "Invitation sent successfully." },
  "page.home.invite_heading": { th: "เชิญสมาชิก", en: "Invite a member" },
  "page.home.projects_heading": { th: "โครงการ", en: "Projects" },
  "page.home.projects_desc": { th: "จัดการและดูโครงการทั้งหมดในเวิร์กสเปซนี้", en: "Manage and view all projects in this workspace." },
  "btn.go_to_projects": { th: "ไปที่โครงการ", en: "Go to projects" },

  // --- role option labels ---
  "role.admin": { th: "Admin", en: "Admin" },
  "role.executive": { th: "Executive", en: "Executive" },
  "role.user": { th: "User", en: "User" },
  "role.owner": { th: "Owner", en: "Owner" },

  // --- accept invite states ---
  "page.accept_invite.title": { th: "ยอมรับคำเชิญ", en: "Accept invitation" },
  "msg.accepting_invite": { th: "กำลังยอมรับคำเชิญ…", en: "Accepting invitation…" },
  "msg.accept_invite_success": { th: "ยอมรับคำเชิญสำเร็จ! กำลังพาคุณไปยัง Workspace…", en: "Invitation accepted! Redirecting to your workspace…" },
  "msg.accept_invite_missing_token": {
    th: "ไม่พบโทเค็นคำเชิญ กรุณาคลิกลิงก์จากอีเมล",
    en: "No invitation token found. Please click the link in your email.",
  },

  // --- nav / projects page ---
  "nav.projects": { th: "โครงการ", en: "Projects" },
  "page.projects.title": { th: "โครงการ", en: "Projects" },
  "page.projects.empty_heading": { th: "ยังไม่มีโครงการ", en: "No projects yet" },
  "page.projects.create_heading": { th: "สร้างโครงการใหม่", en: "Create a new project" },
  "msg.projects_empty_hint": { th: "เริ่มต้นด้วยการสร้างโครงการแรกของคุณ", en: "Get started by creating your first project." },
  "msg.projects_member_readonly": {
    th: "คุณสามารถดูโครงการได้ การสร้างโครงการเป็นสิทธิ์ของเจ้าของหรือผู้ดูแล",
    en: "You can browse projects; creating is for owners/admins.",
  },

  // --- project field labels ---
  "label.project_name": { th: "ชื่อโครงการ", en: "Project name" },
  "label.project_slug": { th: "Slug (ตัวระบุ URL)", en: "Slug (URL identifier)" },
  "label.project_type": { th: "ประเภทโครงการ", en: "Project type" },
  "label.project_status": { th: "สถานะ", en: "Status" },
  "label.requesting_unit": { th: "หน่วยงานผู้ขอ", en: "Requesting unit" },
  "label.description": { th: "รายละเอียด", en: "Description" },
  "label.start_date": { th: "วันที่เริ่ม", en: "Start date" },
  "label.end_date": { th: "วันที่สิ้นสุด", en: "End date" },
  "label.owner": { th: "เจ้าของโครงการ", en: "Owner" },
  "label.created_at": { th: "สร้างเมื่อ", en: "Created" },
  "label.updated_at": { th: "แก้ไขล่าสุด", en: "Updated" },

  // --- buttons / actions ---
  "btn.create_project": { th: "สร้างโครงการ", en: "Create project" },
  "btn.new_project": { th: "+ โครงการใหม่", en: "+ New project" },
  "btn.create_first_project": { th: "สร้างโครงการแรกของคุณ", en: "Create your first project" },
  "btn.open_project": { th: "เปิด", en: "Open" },
  "btn.edit": { th: "แก้ไข", en: "Edit" },
  "btn.save": { th: "บันทึก", en: "Save" },
  "btn.cancel": { th: "ยกเลิก", en: "Cancel" },
  "btn.delete": { th: "ลบ", en: "Delete" },
  "btn.change_status": { th: "เปลี่ยนสถานะ", en: "Change status" },
  "btn.back_to_projects": { th: "กลับไปหน้าโครงการ", en: "Back to projects" },
  "btn.prev": { th: "ก่อนหน้า", en: "Previous" },
  "btn.next": { th: "ถัดไป", en: "Next" },

  // --- filters / pagination ---
  "filter.all": { th: "ทั้งหมด", en: "All" },
  "filter.status": { th: "กรองตามสถานะ", en: "Filter by status" },
  "filter.type": { th: "กรองตามประเภท", en: "Filter by type" },
  "pagination.page_of": { th: "หน้า {page} จาก {total}", en: "Page {page} of {total}" }, // caller MUST .replace — t() does not interpolate

  // --- detail / dialogs ---
  "page.project_detail.not_found": { th: "ไม่พบโครงการนี้ หรือคุณไม่มีสิทธิ์เข้าถึง", en: "Project not found or you don't have access." },
  "dialog.delete.title": { th: "ลบโครงการ?", en: "Delete project?" },
  "dialog.delete.body": { th: "การลบจะนำโครงการนี้ออกจากรายการ คุณแน่ใจหรือไม่?", en: "This will remove the project from your list. Are you sure?" },
  "dialog.change_status.title": { th: "เปลี่ยนสถานะโครงการ?", en: "Change project status?" },
  "msg.status_changed": { th: "เปลี่ยนสถานะเรียบร้อยแล้ว", en: "Status updated." },
  "hint.internal_requesting_unit": { th: "โครงการภายในควรระบุหน่วยงานผู้ขอ", en: "Internal projects should specify a requesting unit." },

  // --- team section (M2 FE-B) ---
  "team.heading": { th: "ทีมงาน", en: "Team" },
  "team.add_member": { th: "เพิ่มสมาชิก", en: "Add member" },
  "team.owner_badge": { th: "เจ้าของ", en: "Owner" },
  "team.owner_locked_hint": {
    th: "การเปลี่ยนเจ้าของทำผ่านการโอนสิทธิ์ ซึ่งจะเปิดให้ใช้ภายหลัง",
    en: "Owner changes happen via ownership transfer, available later.",
  },
  "team.empty": { th: "ยังไม่มีสมาชิกในทีม", en: "No team members yet." },
  "team.no_eligible_members": {
    th: "ไม่มีสมาชิกเวิร์กสเปซที่เพิ่มได้ เชิญสมาชิกใหม่ก่อน",
    en: "No workspace members available to add — invite someone first.",
  },
  "team.invite_member": { th: "เชิญสมาชิก", en: "Invite a member" },
  "label.member": { th: "สมาชิก", en: "Member" },
  "label.member_role": { th: "บทบาทในโครงการ", en: "Project role" },
  "label.select_member": { th: "เลือกสมาชิก", en: "Select member" },
  "dialog.add_member.title": { th: "เพิ่มสมาชิกในโครงการ", en: "Add a project member" },
  "dialog.remove_member.title": { th: "นำสมาชิกออก?", en: "Remove member?" },
  "dialog.remove_member.body": {
    th: "สมาชิกนี้จะถูกนำออกจากโครงการ คุณแน่ใจหรือไม่?",
    en: "This member will be removed from the project. Are you sure?",
  },
  "btn.add": { th: "เพิ่ม", en: "Add" },
  "btn.remove": { th: "นำออก", en: "Remove" },

  // --- project-role labels (5) — pinned to migration 000009 label_th/label_en (lines 72-76) ---
  "project_role.project_owner": { th: "เจ้าของโครงการ", en: "Project Owner" },
  "project_role.project_manager": { th: "ผู้จัดการโครงการ", en: "Project Manager" },
  "project_role.member": { th: "สมาชิก", en: "Member" },
  "project_role.finance": { th: "การเงิน", en: "Finance" },
  "project_role.viewer": { th: "ผู้ดูข้อมูล", en: "Viewer" },

  // --- status labels (8) — pinned to migration 000009 label_th/label_en ---
  "status.draft": { th: "ฉบับร่าง", en: "Draft" },
  "status.planning": { th: "วางแผน", en: "Planning" },
  "status.proposal": { th: "เสนอราคา", en: "Proposal" },
  "status.active": { th: "กำลังดำเนินการ", en: "Active" },
  "status.closing": { th: "กำลังปิดงาน", en: "Closing" },
  "status.maintenance": { th: "ดูแลรักษา", en: "Maintenance" },
  "status.closed": { th: "ปิดโครงการ", en: "Closed" },
  "status.archived": { th: "เก็บถาวร", en: "Archived" },

  // --- type labels (2) — pinned to migration 000009 ---
  "type.internal": { th: "ภายในองค์กร", en: "Internal" },
  "type.client": { th: "งานลูกค้า", en: "Client" },

  // --- new validation error keys (returned by validators) ---
  "project_name_required": { th: "กรุณากรอกชื่อโครงการ", en: "Project name is required." },
  "project_name_too_long": { th: "ชื่อโครงการต้องไม่เกิน 200 ตัวอักษร", en: "Project name must be 200 characters or fewer." },
  "requesting_unit_too_long": { th: "หน่วยงานผู้ขอต้องไม่เกิน 200 ตัวอักษร", en: "Requesting unit must be 200 characters or fewer." },
  "description_too_long": { th: "รายละเอียดต้องไม่เกิน 10000 ตัวอักษร", en: "Description must be 10000 characters or fewer." },
  "invalid_date": { th: "กรุณากรอกวันที่ในรูปแบบ ปปปป-ดด-วว", en: "Please enter a valid date (YYYY-MM-DD)." },
  "end_before_start": { th: "วันที่สิ้นสุดต้องไม่อยู่ก่อนวันที่เริ่ม", en: "End date must not be before the start date." },

  // --- settings / positions (M2 FE-C) ---
  "nav.settings": { th: "ตั้งค่า", en: "Settings" },
  "page.positions.title": { th: "ตำแหน่ง", en: "Positions" },
  "positions.project_heading": { th: "ตำแหน่งในโครงการ", en: "Project positions" },
  "positions.company_heading": { th: "ตำแหน่งในบริษัท", en: "Company positions" },
  "positions.project_desc": {
    th: "หมวกที่สมาชิกสวมในโครงการ (มอบหมายได้หลายตำแหน่งต่อคน)",
    en: "Hats a member wears within a project (a member can hold several).",
  },
  "positions.company_desc": {
    th: "ตำแหน่งงานระดับบริษัทของสมาชิก (หนึ่งตำแหน่งต่อคน)",
    en: "A member's company-wide job title (one per member).",
  },
  "positions.empty": { th: "ยังไม่มีตำแหน่ง — สร้างรายการแรก", en: "No positions yet — create the first one." },
  "positions.add": { th: "เพิ่มตำแหน่ง", en: "Add position" },
  "positions.deprecated_badge": { th: "เลิกใช้แล้ว", en: "Deprecated" },
  "positions.system_badge": { th: "ระบบ", en: "System" },
  "label.position_code": { th: "รหัส (code)", en: "Code" },
  "label.position_label_th": { th: "ชื่อ (ไทย)", en: "Label (TH)" },
  "label.position_label_en": { th: "ชื่อ (อังกฤษ)", en: "Label (EN)" },
  "label.position_description": { th: "คำอธิบาย", en: "Description" },
  "label.position_sort_order": { th: "ลำดับการแสดง", en: "Sort order" },
  "label.position_status": { th: "สถานะ", en: "Status" },
  "status.position.active": { th: "ใช้งาน", en: "Active" },
  "status.position.deprecated": { th: "เลิกใช้", en: "Deprecated" },
  "dialog.create_position.title": { th: "เพิ่มตำแหน่ง", en: "Add position" },
  "dialog.edit_position.title": { th: "แก้ไขตำแหน่ง", en: "Edit position" },
  "dialog.deprecate_position.title": { th: "เลิกใช้ตำแหน่งนี้?", en: "Deprecate this position?" },
  "dialog.deprecate_position.body": {
    th: "ตำแหน่งจะถูกซ่อนจากตัวเลือก แต่ยังคงอยู่กับการมอบหมายเดิม คุณแน่ใจหรือไม่?",
    en: "It will be hidden from pickers but kept on existing assignments. Are you sure?",
  },
  "btn.deprecate": { th: "เลิกใช้", en: "Deprecate" },
  "msg.positions_readonly": {
    th: "คุณมีสิทธิ์ดูอย่างเดียว การจัดการตำแหน่งเป็นสิทธิ์ของเจ้าของหรือผู้ดูแล",
    en: "You have read-only access; managing positions is for owners/admins.",
  },
  "position_code_required": { th: "กรุณากรอกรหัสตำแหน่ง", en: "Position code is required." },
  "position_code_invalid": {
    th: "รหัสต้องเป็นตัวพิมพ์เล็ก ตัวเลข หรือ _ ขึ้นต้นด้วยตัวอักษร ความยาว 2–63 ตัว",
    en: "Code must be lowercase letters, digits, or underscores, start with a letter, 2–63 chars.",
  },
  "position_label_required": { th: "กรุณากรอกชื่อตำแหน่ง", en: "Label is required." },
  "position_label_too_long": { th: "ชื่อตำแหน่งต้องไม่เกิน 200 ตัวอักษร", en: "Label must be 200 characters or fewer." },
  "position_sort_order_invalid": { th: "ลำดับต้องเป็นจำนวนเต็มตั้งแต่ 0 ขึ้นไป", en: "Sort order must be an integer of 0 or greater." },

  // --- positions on project detail (M2 FE-C2) ---
  "positions.member_heading": { th: "ตำแหน่งของสมาชิก", en: "Member positions" },
  "positions.company_position_label": { th: "ตำแหน่งในบริษัท", en: "Company position" },
  "positions.company_position_hint": {
    th: "ระดับเวิร์กสเปซ — เปลี่ยนแล้วมีผลกับสมาชิกในทุกโครงการ",
    en: "Workspace-level — changing this affects the member across all projects.",
  },
  "positions.company_position_none": { th: "— ไม่ระบุ —", en: "— None —" },
  "positions.member_none": { th: "ยังไม่มีตำแหน่ง", en: "No positions" },
  "positions.no_active_masters": {
    th: "ยังไม่มีตำแหน่งโครงการให้เลือก สร้างก่อนที่หน้าตั้งค่า",
    en: "No project positions available yet — create them in Settings first.",
  },
  "positions.go_to_settings": { th: "ไปที่หน้าตั้งค่า", en: "Go to Settings" },
  "label.select_position": { th: "เลือกตำแหน่ง", en: "Select position" },
  "dialog.assign_position.title": { th: "เพิ่มตำแหน่งในโครงการ", en: "Add project position" },
  "dialog.unassign_position.title": { th: "นำตำแหน่งออก?", en: "Remove this position?" },
  "dialog.unassign_position.body": {
    th: "ตำแหน่งนี้จะถูกนำออกจากสมาชิก คุณแน่ใจหรือไม่?",
    en: "This position will be removed from the member. Are you sure?",
  },
};

export function t(key: string, lang: Lang): string {
  const entry = UI_STRINGS[key];
  if (!entry) return key;
  return entry[lang] ?? key;
}

// ---------------------------------------------------------------------------
// API error code → bilingual message
// ---------------------------------------------------------------------------

const ERROR_MESSAGES: Record<string, Record<Lang, string>> = {
  "auth.email_taken": {
    th: "อีเมลนี้ถูกใช้งานแล้ว",
    en: "This email is already in use.",
  },
  "auth.invalid_credentials": {
    th: "อีเมลหรือรหัสผ่านไม่ถูกต้อง",
    en: "Invalid email or password.",
  },
  "auth.account_locked": {
    th: "บัญชีของคุณถูกล็อค กรุณาติดต่อผู้ดูแลระบบ",
    en: "Your account has been locked. Please contact support.",
  },
  "auth.email_not_verified": {
    th: "บัญชีของคุณยังไม่ได้ยืนยันอีเมล",
    en: "Your email address has not been verified.",
  },
  "auth.token_invalid": {
    th: "ลิงก์ไม่ถูกต้องหรือใช้งานไปแล้ว",
    en: "This link is invalid or has already been used.",
  },
  "auth.token_expired": {
    th: "ลิงก์หมดอายุแล้ว กรุณาขอลิงก์ใหม่",
    en: "This link has expired. Please request a new one.",
  },
  "auth.unauthenticated": {
    th: "กรุณาเข้าสู่ระบบก่อน",
    en: "Please log in to continue.",
  },
  "validation.invalid_input": {
    th: "ข้อมูลที่กรอกไม่ถูกต้อง กรุณาตรวจสอบและลองอีกครั้ง",
    en: "Some fields are invalid. Please check and try again.",
  },
  "workspace.slug_reserved": {
    th: "Slug นี้ถูกสงวนไว้ กรุณาเลือก slug อื่น",
    en: "This slug is reserved. Please choose a different one.",
  },
  "workspace.slug_taken": {
    th: "Slug นี้ถูกใช้งานแล้ว กรุณาเลือก slug อื่น",
    en: "This slug is already taken. Please choose a different one.",
  },
  "tenant.workspace_required": {
    th: "กรุณาระบุ workspace",
    en: "A workspace is required.",
  },
  "tenant.workspace_not_found": {
    th: "ไม่พบ workspace นี้",
    en: "Workspace not found.",
  },
  "tenant.forbidden": {
    th: "คุณไม่มีสิทธิ์เข้าถึง workspace นี้",
    en: "You do not have access to this workspace.",
  },
  "tenant.permission_denied": {
    th: "คุณไม่มีสิทธิ์ดำเนินการนี้",
    en: "You do not have permission to perform this action.",
  },
  "invitation.role_not_allowed": {
    th: "บทบาทนี้ไม่สามารถเชิญได้",
    en: "This role cannot be invited.",
  },
  "invitation.already_member": {
    th: "ผู้ใช้นี้เป็นสมาชิกของ workspace แล้ว",
    en: "This user is already a member of the workspace.",
  },
  "invitation.already_pending": {
    th: "มีคำเชิญที่ยังไม่ได้รับการยืนยันสำหรับอีเมลนี้อยู่แล้ว",
    en: "There is already a pending invitation for this email.",
  },
  "invitation.invalid": {
    th: "คำเชิญนี้ไม่ถูกต้องหรือใช้งานไปแล้ว",
    en: "This invitation is invalid or has already been used.",
  },
  "invitation.expired": {
    th: "คำเชิญนี้หมดอายุแล้ว กรุณาขอคำเชิญใหม่",
    en: "This invitation has expired. Please request a new one.",
  },
  "invitation.email_mismatch": {
    th: "คำเชิญนี้ถูกส่งไปยังอีเมลอื่น กรุณาเข้าสู่ระบบด้วยอีเมลที่ถูกเชิญ",
    en: "This invitation was sent to a different email address. Please log in with the invited email.",
  },

  // --- project codes (M2 FE-A) ---
  "project.slug_taken": {
    th: "Slug นี้ถูกใช้งานแล้วในโครงการอื่น กรุณาเลือก slug อื่น",
    en: "This slug is already used by another project. Please choose a different one.",
  },
  "project.invalid_master_code": {
    th: "รหัสสถานะหรือประเภทไม่ถูกต้อง",
    en: "Invalid status or type code.",
  },
  "project.not_found": {
    th: "ไม่พบโครงการนี้",
    en: "Project not found.",
  },

  // --- project member codes (M2 FE-B) ---
  "project_member.membership_not_eligible": {
    th: "สมาชิกที่เลือกไม่สามารถเพิ่มได้",
    en: "The selected member is not eligible to add.",
  },
  "project_member.invalid_role_code": {
    th: "บทบาทในโครงการไม่ถูกต้อง",
    en: "Invalid project role.",
  },
  "project_member.owner_immutable": {
    th: "ไม่สามารถแก้ไขเจ้าของโครงการนอกการโอนสิทธิ์",
    en: "The project owner cannot be modified outside ownership transfer.",
  },
  "project_member.already_member": {
    th: "ผู้ใช้นี้เป็นสมาชิกของโครงการอยู่แล้ว",
    en: "This user is already a member of the project.",
  },
  "project_member.not_found": {
    th: "ไม่พบสมาชิกนี้",
    en: "Member not found.",
  },

  // --- positions codes (M2 FE-C) — same message for both project_/company_ families ---
  "project_position.code_taken": {
    th: "รหัสนี้ถูกใช้แล้วในเวิร์กสเปซ",
    en: "This code already exists in this workspace.",
  },
  "company_position.code_taken": {
    th: "รหัสนี้ถูกใช้แล้วในเวิร์กสเปซ",
    en: "This code already exists in this workspace.",
  },
  "project_position.code_immutable": {
    th: "ไม่สามารถเปลี่ยนรหัสตำแหน่งได้",
    en: "The position code cannot be changed.",
  },
  "company_position.code_immutable": {
    th: "ไม่สามารถเปลี่ยนรหัสตำแหน่งได้",
    en: "The position code cannot be changed.",
  },
  "project_position.system_immutable": {
    th: "ไม่สามารถแก้ไขตำแหน่งของระบบได้",
    en: "System positions cannot be modified.",
  },
  "company_position.system_immutable": {
    th: "ไม่สามารถแก้ไขตำแหน่งของระบบได้",
    en: "System positions cannot be modified.",
  },
  "project_position.invalid_status": {
    th: "สถานะไม่ถูกต้อง",
    en: "Invalid status.",
  },
  "company_position.invalid_status": {
    th: "สถานะไม่ถูกต้อง",
    en: "Invalid status.",
  },
  "project_position.not_found": {
    th: "ไม่พบตำแหน่งนี้",
    en: "Position not found.",
  },
  "company_position.not_found": {
    th: "ไม่พบตำแหน่งนี้",
    en: "Position not found.",
  },

  // --- member position assignment codes (M2 FE-C2) ---
  "project_member_position.already_assigned": {
    th: "สมาชิกมีตำแหน่งนี้อยู่แล้ว",
    en: "This position is already assigned to the member.",
  },
  "project_member_position.invalid_position_code": {
    th: "รหัสตำแหน่งไม่ถูกต้องหรือถูกเลิกใช้แล้ว",
    en: "Invalid or deprecated position code.",
  },
  "project_member_position.member_removed": {
    th: "สมาชิกถูกนำออกจากโครงการแล้ว",
    en: "This member has been removed from the project.",
  },
  "project_member_position.member_not_found": {
    th: "ไม่พบสมาชิกในโครงการ",
    en: "Project member not found.",
  },
  "project_member_position.not_found": {
    th: "ไม่พบตำแหน่งที่กำหนดให้",
    en: "Assignment not found.",
  },
  "company_position.invalid_position_code": {
    th: "รหัสตำแหน่งบริษัทไม่ถูกต้องหรือถูกเลิกใช้แล้ว",
    en: "Invalid or deprecated company position code.",
  },
  "company_position.membership_not_found": {
    th: "ไม่พบสมาชิกในเวิร์กสเปซ",
    en: "Workspace membership not found.",
  },
  "internal.unexpected": {
    th: "เกิดข้อผิดพลาดภายในระบบ กรุณาลองอีกครั้ง",
    en: "An unexpected server error occurred. Please try again.",
  },
};

const FALLBACK: Record<Lang, string> = {
  th: "เกิดข้อผิดพลาด กรุณาลองอีกครั้ง",
  en: "An error occurred. Please try again.",
};

export function errorMessage(code: string, lang: Lang): string {
  const entry = ERROR_MESSAGES[code];
  if (!entry) return FALLBACK[lang];
  return entry[lang] ?? FALLBACK[lang];
}
