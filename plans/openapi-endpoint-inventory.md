# OpenAPI Endpoint Inventory

- Source: `../tmcopilot-project/backend/docs/swagger/swagger.json`
- SHA256: `a6107d1532876814a348c347fdf6a2fd1fbba8e42097655d030ba2c9e24ea12d`
- Endpoints: `149`

| Coverage | Method | Path | Tags | Summary |
|---|---|---|---|---|
| raw | POST | `/agent/approvals/{id}/approve` | agent | Approve agent action |
| raw | POST | `/agent/approvals/{id}/reject` | agent | Reject agent action |
| raw | DELETE | `/agent/runs/{id}` | agent | Cancel agent run |
| raw | GET | `/agent/runs/{id}` | agent | Get agent run |
| raw | GET | `/agent/threads` | agent | List agent threads |
| raw | POST | `/agent/threads` | agent | Create agent thread |
| raw | DELETE | `/agent/threads/{id}` | agent | Delete agent thread |
| raw | GET | `/agent/threads/{id}` | agent | Get agent thread |
| raw | GET | `/agent/threads/{id}/events` | agent | Stream agent events |
| raw | POST | `/agent/threads/{id}/events/ticket` | agent | Create agent SSE ticket |
| raw | POST | `/agent/threads/{id}/runs` | agent | Start agent run |
| raw | GET | `/auth/admin/invitations` | admin | List invitation tokens (admin) |
| raw | POST | `/auth/admin/invitations` | admin | Create an invitation link (admin) |
| raw | GET | `/auth/admin/message-delivery-events` | admin | List email delivery events (admin) |
| raw | GET | `/auth/admin/service-requests` | admin | List service requests (admin) |
| raw | PATCH | `/auth/admin/service-requests/{id}` | admin | Update service request status (admin) |
| raw | GET | `/auth/admin/settings` | admin | Get system settings (admin) |
| raw | PUT | `/auth/admin/settings` | admin | Update system settings (admin) |
| raw | GET | `/auth/admin/users` | admin | List all users (admin) |
| raw | POST | `/auth/admin/users` | admin | Create user (admin) |
| raw | PUT | `/auth/admin/users/{id}/admin-level` | admin | Update user admin level (admin) |
| raw | PUT | `/auth/admin/users/{id}/status` | admin | Update user status (admin) |
| raw | GET | `/auth/admin/workspaces` | admin | List all workspaces (admin) |
| raw | POST | `/auth/admin/workspaces/{workspace_id}/members` | admin | Add workspace member (admin) |
| typed | GET | `/auth/api-keys` | auth | List API keys |
| typed | POST | `/auth/api-keys` | auth | Create an API key |
| typed | DELETE | `/auth/api-keys/{id}` | auth | Revoke an API key |
| typed | GET | `/auth/collaborators` | auth | List collaborators |
| typed | POST | `/auth/collaborators/invitations` | auth | Create collaborator invitation |
| typed | DELETE | `/auth/collaborators/invitations/{id}` | auth | Delete collaborator invitation |
| typed | POST | `/auth/collaborators/invitations/{token}/accept` | auth | Accept collaborator invitation |
| typed | DELETE | `/auth/collaborators/{id}` | auth | Remove collaborator |
| typed | PUT | `/auth/collaborators/{id}/role` | auth | Update collaborator role |
| raw | GET | `/auth/invitation/{token}` | auth | Validate an invitation token |
| raw-ready | POST | `/auth/login` | auth | Log in |
| typed | POST | `/auth/logout` | auth | Log out |
| typed | GET | `/auth/me` | auth | Get current user profile |
| raw | PUT | `/auth/me` | auth | Update current user profile |
| raw | PUT | `/auth/me/password` | auth | Change password |
| typed | GET | `/auth/notification-preferences` | auth | Get notification preferences |
| typed | PUT | `/auth/notification-preferences` | auth | Update notification preferences |
| raw | GET | `/auth/notifications` | auth | List notifications |
| raw | PUT | `/auth/notifications/read` | auth | Mark notifications read |
| raw | PUT | `/auth/notifications/read-all` | auth | Mark all notifications read |
| raw | DELETE | `/auth/notifications/{id}` | auth | Dismiss notification |
| raw | POST | `/auth/refresh` | auth | Refresh tokens |
| raw | POST | `/auth/register` | auth | Register a new user |
| raw | POST | `/auth/register/verification-code` | auth | Send registration verification code |
| raw | GET | `/auth/registration-status` | auth | Check registration status |
| typed | GET | `/auth/ui-settings` | auth | Get dashboard UI settings |
| raw | PUT | `/auth/workspace` | auth | Rename workspace |
| typed | GET | `/auth/workspaces` | auth | List accessible workspaces |
| typed | POST | `/common-law/max-similarity` | common-law | Get max common law similarity |
| typed | POST | `/common-law/search/app-store` | common-law | App store search for common law |
| typed | POST | `/common-law/search/ecommerce/handle` | common-law | E-commerce handle search for common law |
| typed | POST | `/common-law/search/google/text` | common-law | Google text search for common law |
| typed | POST | `/common-law/search/social/handle` | common-law | Social handle search for common law |
| typed | POST | `/common-law/search/social/text` | common-law | Social network text search for common law |
| typed | GET | `/competitors` | competitors | List competitors |
| typed | GET | `/competitors/activities` | competitors | List competitor activities |
| typed | GET | `/competitors/reports` | competitors | List competitor reports |
| typed | POST | `/domain/max-similarity` | domain | Get max domain name similarity |
| typed | POST | `/domain/search` | domain | Search domain names by keyword |
| typed | GET | `/files` | files | List files |
| typed | POST | `/files/presign` | files | Create file upload URL |
| typed | GET | `/gap-analyses` | gap-analysis | List gap analyses |
| typed | POST | `/gap-analyses` | gap-analysis | Create gap analysis |
| typed | GET | `/gap-analyses/shares/{token}` | gap-analysis | Get shared gap analysis |
| typed | DELETE | `/gap-analyses/{id}` | gap-analysis | Delete gap analysis |
| typed | GET | `/gap-analyses/{id}` | gap-analysis | Get gap analysis |
| typed | GET | `/gap-analyses/{id}/reports` | gap-analysis | List gap analysis reports |
| typed | POST | `/gap-analyses/{id}/reports/generate` | gap-analysis | Generate gap analysis report |
| typed | GET | `/gap-analyses/{id}/results` | gap-analysis | Get gap analysis results |
| typed | POST | `/gap-analyses/{id}/run` | gap-analysis | Run gap analysis |
| typed | POST | `/gap-analyses/{id}/share` | gap-analysis | Create gap analysis share |
| typed | GET | `/gap-analyses/{id}/shares` | gap-analysis | List gap analysis shares |
| typed | DELETE | `/gap-analyses/{id}/shares/{token}` | gap-analysis | Delete gap analysis share |
| typed | GET | `/portfolio/actions/cbp` | portfolio-action | List CBP recordations |
| raw | GET | `/portfolio/actions/cbp/service-requests` | portfolio-action | List CBP recordation service requests |
| raw | POST | `/portfolio/actions/cbp/service-requests` | portfolio-action | Submit a CBP recordation service request |
| typed | GET | `/portfolio/actions/cbp/summary` | portfolio-action | Get CBP recordation summary |
| typed | GET | `/portfolio/actions/conflict` | portfolio-action | List conflict actions |
| raw | GET | `/portfolio/actions/conflict/groups` | portfolio-action | List grouped conflict actions |
| typed | GET | `/portfolio/actions/conflict/summary` | portfolio-action | Get conflict action summary |
| typed | GET | `/portfolio/actions/office` | portfolio-action | List office actions |
| raw | GET | `/portfolio/actions/office/deadlines` | portfolio-action | Get upcoming deadlines |
| typed | GET | `/portfolio/actions/office/summary` | portfolio-action | Get office action summary |
| typed | GET | `/portfolio/activity` | portfolio-action | List portfolio activity |
| raw | GET | `/portfolio/tasks` | portfolio-task | List portfolio worker tasks |
| raw | GET | `/portfolio/tasks/latest-sync` | portfolio-task | Get latest portfolio task sync |
| raw | GET | `/portfolio/tasks/stats` | portfolio-task | Get portfolio task stats |
| raw | GET | `/portfolio/tasks/{taskId}` | portfolio-task | Get portfolio worker task |
| typed | GET | `/portfolio/trademark-groups` | portfolio-trademark | List trademark groups |
| typed | PUT | `/portfolio/trademark-groups/{groupId}/monitor/toggle` | portfolio-trademark | Toggle trademark group monitor type |
| typed | PUT | `/portfolio/trademark-monitor` | portfolio-trademark | Batch update monitor config |
| typed | PUT | `/portfolio/trademark-monitor/toggle` | portfolio-trademark | Batch toggle trademark monitor type |
| raw | DELETE | `/portfolio/trademarks` | portfolio-trademark | Delete portfolio trademarks |
| raw | GET | `/portfolio/trademarks` | portfolio-trademark | List portfolio trademarks |
| raw | POST | `/portfolio/trademarks` | portfolio-trademark | Batch create portfolio trademarks |
| typed | GET | `/portfolio/trademarks/counts` | portfolio-trademark | Get trademark counts |
| typed | POST | `/portfolio/trademarks/import` | portfolio-trademark | Import trademarks by lawyer/owner names |
| typed | POST | `/portfolio/trademarks/import/preview` | portfolio-trademark | Preview portfolio trademark import |
| typed | GET | `/portfolio/trademarks/monitored` | portfolio-trademark | List monitored trademarks |
| typed | GET | `/portfolio/trademarks/search` | portfolio-trademark | Search portfolio trademarks |
| typed | GET | `/portfolio/trademarks/{trademarkId}` | portfolio-trademark | Get a portfolio trademark |
| typed | PUT | `/portfolio/trademarks/{trademarkId}` | portfolio-trademark | Update a portfolio trademark |
| raw | GET | `/portfolio/trademarks/{trademarkId}/conflict-actions` | portfolio-action | List conflict actions by trademark |
| raw | GET | `/portfolio/trademarks/{trademarkId}/conflict-actions/{id}` | portfolio-action | Get a conflict action |
| raw | PUT | `/portfolio/trademarks/{trademarkId}/conflict-actions/{id}/status` | portfolio-action | Update conflict action status |
| typed | GET | `/portfolio/trademarks/{trademarkId}/metadata` | portfolio-trademark | Get manual portfolio trademark metadata |
| typed | PUT | `/portfolio/trademarks/{trademarkId}/metadata` | portfolio-trademark | Update manual portfolio trademark metadata |
| typed | PUT | `/portfolio/trademarks/{trademarkId}/monitor` | portfolio-trademark | Update trademark monitor config |
| raw | GET | `/portfolio/trademarks/{trademarkId}/office-actions` | portfolio-action | List office actions by trademark |
| raw | GET | `/portfolio/trademarks/{trademarkId}/office-actions/{id}` | portfolio-action | Get an office action |
| raw | PUT | `/portfolio/trademarks/{trademarkId}/office-actions/{id}/status` | portfolio-action | Update office action status |
| typed | POST | `/trademark/detail` | trademark | Get trademark details |
| typed | POST | `/trademark/image/task` | trademark | Create image search task |
| typed | POST | `/trademark/image/task/result` | trademark | Get image search task result |
| typed | GET | `/trademark/image/task/{id}/result` | trademark | Get image search task result |
| typed | GET | `/trademark/lawyer/contact` | trademark | Get lawyer contact information |
| typed | GET | `/trademark/lawyer/ranking` | trademark | Get lawyer ranking list from Doris |
| typed | GET | `/trademark/lawyer/search` | trademark | Search lawyers by name or advanced filters |
| raw | POST | `/trademark/max-similarity` | trademark | Get max trademark similarity per analysis type |
| typed | POST | `/trademark/office-action/search` | trademark | Search Office Actions |
| typed | GET | `/trademark/office-action/uspto/document` | trademark | Get USPTO document |
| typed | GET | `/trademark/owner/ranking` | trademark | Get owner ranking list from Doris |
| typed | GET | `/trademark/owner/search` | trademark | Search brand owners by name |
| typed | POST | `/trademark/search` | trademark | Search US trademarks by text |
| raw | POST | `/trademark/search/runs/{runId}/shares` | trademark | Create trademark search share |
| raw | GET | `/trademark/search/shares` | trademark | List trademark search shares |
| raw | DELETE | `/trademark/search/shares/{token}` | trademark | Delete trademark search share |
| raw | GET | `/trademark/search/shares/{token}` | trademark | Get trademark search share |
| raw | PUT | `/trademark/search/shares/{token}` | trademark | Update trademark search share content |
| raw | PUT | `/trademark/search/shares/{token}/active` | trademark | Update trademark search share active state |
| typed | POST | `/trademark/search/summary` | trademark | Generate trademark search summary |
| typed | GET | `/trademark/search/tips` | trademark | Search autocomplete tips |
| typed | POST | `/trademark/ttab/search` | trademark | Search TTAB cases |
| typed | GET | `/trademark/ttab/{case_number}` | trademark | Get TTAB case details |
| raw | GET | `/trademark/wide-table/brand-owners/{graphId}` | trademark | Get brand owner info |
| raw | POST | `/trademark/wide-table/brand-owners/{graphId}/law-firms` | trademark | List brand owner law firms |
| typed | POST | `/trademark/wide-table/brand-owners/{graphId}/lawsuits` | trademark | List brand owner lawsuits |
| raw | POST | `/trademark/wide-table/brand-owners/{graphId}/trademarks` | trademark | List brand owner trademarks |
| typed | POST | `/trademark/wide-table/lawsuits` | trademark | Search lawsuits |
| typed | GET | `/trademark/wide-table/lawsuits/{caseNumber}` | trademark | Get lawsuit detail |
| typed | GET | `/trademark/wide-table/lawyers/{graphId}` | trademark | Get lawyer info |
| typed | POST | `/trademark/wide-table/lawyers/{graphId}/law-firms` | trademark | List lawyer law firms |
| typed | POST | `/trademark/wide-table/lawyers/{graphId}/lawsuits` | trademark | List lawyer lawsuits |
| typed | POST | `/trademark/wide-table/lawyers/{graphId}/trademarks` | trademark | List lawyer trademarks |
| typed | POST | `/upload/presign` | upload | Create S3 presigned upload URL |
