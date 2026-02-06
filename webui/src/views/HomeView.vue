<script>
export default {
	data() {
		return {
			errormsg: null,
			token: localStorage.getItem('token') || '',
			usernameInput: '',
			me: null,
			users: [],
			conversations: [],
			selectedConversationId: null,
			selectedConversation: null,
			messages: [],
			text: '',
			image: '',
			imageFileName: '',
			replyToId: null,
			reactionEmoji: '🔥',
			groupName: '',
			groupMembersCSV: '',
			profileName: '',
			profilePhoto: '',
			groupEditName: '',
			groupEditPhoto: '',
			pollHandle: null,
		}
	},
		methods: {
			avatarPhoto(entity) {
				if (!entity || !entity.photo) return ''
				return entity.photo
			},
			api() {
				return { headers: this.token ? { Authorization: `Bearer ${this.token}` } : {} }
			},
		initials(name) {
			if (!name) return '?'
			return name.split(' ').map(v => v[0]).join('').slice(0, 2).toUpperCase()
		},
		msgPreview(msg) {
			if (!msg) return 'Nessun messaggio'
			if (msg.text) return msg.text.length > 32 ? `${msg.text.slice(0, 32)}…` : msg.text
			if (msg.image) return '📷 Immagine'
			return 'Messaggio'
		},
		formatClock(ts) {
			if (!ts) return ''
			const d = new Date(ts)
			if (Number.isNaN(d.getTime())) return ''
			return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
		},
		async login() {
			this.errormsg = null
			try {
				const r = await this.$axios.post('/api/login', { username: this.usernameInput })
				this.token = r.data.token
				localStorage.setItem('token', this.token)
				await this.refreshAll()
				this.startPolling()
			} catch (e) {
				this.errormsg = e.response?.data?.error || e.toString()
			}
		},
		async logout() {
			try { await this.$axios.post('/api/logout', {}, this.api()) } catch (_) {}
			if (this.pollHandle) clearInterval(this.pollHandle)
			this.token = ''
			this.me = null
			this.messages = []
			this.selectedConversation = null
			this.selectedConversationId = null
			localStorage.removeItem('token')
		},
		startPolling() {
			if (this.pollHandle) clearInterval(this.pollHandle)
			this.pollHandle = setInterval(async () => {
				try { await this.refreshAll() } catch (_) {}
			}, 500)
		},
		async refreshAll() {
			if (!this.token) return
			const [me, users, conversations] = await Promise.all([
				this.$axios.get('/api/me', this.api()),
				this.$axios.get('/api/users', this.api()),
				this.$axios.get('/api/conversations', this.api()),
			])
			this.me = me.data
			this.profileName = me.data.displayName
			this.profilePhoto = me.data.photo
			this.users = users.data
			this.conversations = conversations.data
			if (this.selectedConversationId) {
				await this.openConversation(this.selectedConversationId)
			}
		},
		async openConversation(id) {
			this.selectedConversationId = id
			const r = await this.$axios.get(`/api/conversation/${id}`, this.api())
			this.selectedConversation = r.data.conversation
			this.messages = r.data.messages
			for (const m of this.messages) {
				if (m.sender.id !== this.me.id) {
					await this.$axios.post(`/api/messages/${m.id}/read`, {}, this.api())
				}
			}
		},
		async createDirect(username) {
			await this.$axios.post('/api/conversations/direct', { username }, this.api())
			await this.refreshAll()
		},
		async createGroup() {
			const members = this.groupMembersCSV.split(',').map(v => v.trim()).filter(Boolean)
			await this.$axios.post('/api/conversations/group', { name: this.groupName, members }, this.api())
			this.groupName = ''
			this.groupMembersCSV = ''
			await this.refreshAll()
		},
		async send() {
			if (!this.selectedConversationId) return
			await this.$axios.post(`/api/conversation/${this.selectedConversationId}/messages`, {
				text: this.text,
				image: this.image,
				replyToId: this.replyToId,
			}, this.api())
			this.text = ''
			this.image = ''
			this.replyToId = null
			await this.openConversation(this.selectedConversationId)
		},
		async react(msg) {
			await this.$axios.post(`/api/messages/${msg.id}/reaction`, { emoji: this.reactionEmoji }, this.api())
			await this.openConversation(this.selectedConversationId)
		},
		async unreact(msg) {
			await this.$axios.delete(`/api/messages/${msg.id}/reaction`, this.api())
			await this.openConversation(this.selectedConversationId)
		},
		async forward(msg, toConversationId) {
			if (!toConversationId) return
			await this.$axios.post(`/api/messages/${msg.id}/forward`, { toConversationId }, this.api())
			await this.openConversation(this.selectedConversationId)
		},
		async leaveGroup() {
			await this.$axios.post(`/api/conversation/${this.selectedConversationId}/leave`, {}, this.api())
			this.selectedConversationId = null
			this.selectedConversation = null
			this.messages = []
			await this.refreshAll()
		},
		async addMember(username) {
			if (!username) return
			await this.$axios.post(`/api/conversation/${this.selectedConversationId}/add`, { username }, this.api())
			await this.openConversation(this.selectedConversationId)
		},
		async updateProfile() {
			await this.$axios.put('/api/me', {
				displayName: this.profileName,
				photo: this.profilePhoto,
			}, this.api())
			await this.refreshAll()
		},
		async updateGroup() {
			await this.$axios.put(`/api/conversation/${this.selectedConversationId}`, {
				name: this.groupEditName,
				photo: this.groupEditPhoto,
			}, this.api())
			await this.openConversation(this.selectedConversationId)
			await this.refreshAll()
		},
		onImageFileChange(e) {
			const file = e.target.files && e.target.files[0]
			if (!file) return
			this.imageFileName = file.name
			const reader = new FileReader()
			reader.onload = () => {
				this.image = typeof reader.result === 'string' ? reader.result : ''
			}
			reader.readAsDataURL(file)
		},
		clearImage() {
			this.image = ''
			this.imageFileName = ''
		},
		replyPreview(replyToId) {
			const ref = this.messages.find(m => m.id === replyToId)
			if (!ref) return `#${replyToId}`
			const textSnippet = ref.text ? (ref.text.length > 34 ? `${ref.text.slice(0, 34)}…` : ref.text) : ''
			const imageIcon = ref.image ? ' 🖼️' : ''
			return `${textSnippet}${imageIcon}`.trim() || `#${replyToId}`
		},
		readChecks(msg) {
			if (!this.selectedConversation || msg.sender.id !== this.me?.id) return ''
			const recipients = this.selectedConversation.members.filter(m => m.id !== this.me.id)
			if (recipients.length === 0) return ''
			const readByRecipients = recipients.filter(m => msg.readBy.includes(m.id)).length
			if (readByRecipients === recipients.length) return '✓✓'
			return '✓'
		},
		isMine(msg) {
			return msg.sender.id === this.me?.id
		}
	},
	async mounted() {
		if (this.token) {
			try {
				await this.refreshAll()
				this.startPolling()
			} catch (_) {
				this.logout()
			}
		}
	}
}
</script>

<template>
	<div class="wasatx-root">
		<ErrorMsg v-if="errormsg" :msg="errormsg" />

		<div v-if="!token" class="login-shell">
			<div class="login-panel">
				<div class="brand-mark">WA</div>
				<h1>Accedi a WASAtext</h1>
				<p>Chat sicura multiutente. Tema completamente nero e blu scuro.</p>
				<input v-model="usernameInput" class="tx-input" placeholder="username" />
				<button class="tx-btn tx-btn-primary" @click="login">Accedi</button>
				<div class="login-note">⚠️ Chiunque conosca il tuo username può accedere.</div>
			</div>
		</div>

		<div v-else class="chat-screen">
			<header class="main-topbar">
				<div class="top-brand">WASAtext • Dark Blue Edition</div>
				<div class="top-meta">Online come <b>{{ me?.displayName }}</b> • aggiornamento automatico attivo</div>
			</header>

			<div class="chat-shell">
				<aside class="left-rail">
						<div class="me-card">
							<div class="avatar">
								<img v-if="avatarPhoto(me)" :src="avatarPhoto(me)" alt="Foto profilo" />
								<span v-else>{{ initials(me?.displayName || me?.username) }}</span>
							</div>
						<div>
							<div class="me-name">{{ me?.displayName }}</div>
							<div class="me-user">@{{ me?.username }}</div>
						</div>
					</div>

					<div class="panel-card">
						<div class="section-title">Profilo</div>
						<input class="tx-input small" v-model="profileName" placeholder="Nuovo nome" />
						<input class="tx-input small" v-model="profilePhoto" placeholder="URL/base64 foto" />
						<div class="stack-row">
							<button class="tx-btn" @click="updateProfile">Aggiorna</button>
							<button class="tx-btn tx-btn-danger" @click="logout">Logout</button>
						</div>
					</div>

					<div class="panel-card">
						<div class="section-title">Nuovo gruppo</div>
						<input class="tx-input small" v-model="groupName" placeholder="Nome gruppo" />
						<input class="tx-input small" v-model="groupMembersCSV" placeholder="membri: b,c" />
						<button class="tx-btn tx-btn-primary" @click="createGroup">Crea gruppo</button>
					</div>

					<div class="panel-card">
						<div class="section-title">Contatti</div>
						<div class="user-list">
							<div v-for="u in users.filter(x => x.id !== me?.id)" :key="u.id" class="user-row">
								<div class="avatar tiny">
									<img v-if="avatarPhoto(u)" :src="avatarPhoto(u)" alt="Foto profilo" />
									<span v-else>{{ initials(u.displayName) }}</span>
								</div>
								<div class="user-meta">
									<div class="user-name">{{ u.displayName }}</div>
									<div class="user-sub">@{{ u.username }}</div>
								</div>
								<button class="tx-btn mini" @click="createDirect(u.username)">Apri chat</button>
							</div>
						</div>
					</div>
				</aside>

				<section class="conversation-rail">
					<div class="rail-head">Conversazioni</div>
					<div class="conv-list">
						<div
							v-for="c in conversations"
							:key="c.id"
							class="conv-row"
							:class="{ active: selectedConversationId === c.id }"
							@click="openConversation(c.id)">
							<div class="avatar tiny">
								<img v-if="avatarPhoto(c)" :src="avatarPhoto(c)" alt="Foto chat" />
								<span v-else>{{ initials(c.name || 'C') }}</span>
							</div>
							<div class="conv-meta">
								<div class="conv-name">{{ c.name || ('Chat #' + c.id) }}</div>
								<div class="conv-preview">{{ msgPreview(c.lastMessage) }}</div>
							</div>
							<div class="conv-time">{{ formatClock(c.lastMessage?.createdAt) }}</div>
						</div>
					</div>
				</section>

				<main class="chat-main">
					<div v-if="selectedConversation" class="chat-wrap">
						<header class="chat-head">
							<div class="chat-title">
								<div class="avatar tiny">
									<img v-if="avatarPhoto(selectedConversation)" :src="avatarPhoto(selectedConversation)" alt="Foto chat" />
									<span v-else>{{ initials(selectedConversation.name) }}</span>
								</div>
								<div>
									<div class="title-main">{{ selectedConversation.name }}</div>
									<div class="muted">{{ selectedConversation.members?.length || 0 }} partecipanti</div>
								</div>
							</div>
							<div class="chat-actions" v-if="selectedConversation.isGroup">
								<input class="tx-input small" v-model="groupEditName" placeholder="Nome gruppo" />
								<input class="tx-input small" v-model="groupEditPhoto" placeholder="Foto gruppo" />
								<select class="tx-input small" @change="addMember($event.target.value)">
									<option value="">Aggiungi membro…</option>
									<option v-for="u in users" :key="u.id" :value="u.username">{{ u.username }}</option>
								</select>
								<button class="tx-btn mini" @click="updateGroup">Aggiorna</button>
								<button class="tx-btn tx-btn-danger mini" @click="leaveGroup">Esci</button>
							</div>
						</header>

						<section class="msg-scroll">
							<div v-for="m in messages" :key="m.id" class="msg-row" :class="{ mine: isMine(m) }">
								<div class="msg-bubble">
									<div class="msg-top">
										<b>{{ m.sender.displayName }}</b>
										<span class="muted">{{ formatClock(m.createdAt) }} {{ readChecks(m) }}</span>
									</div>
									<div v-if="m.forwarded" class="fwd-label">Inoltrato</div>
									<div v-if="m.replyToId" class="reply-label">↪ {{ replyPreview(m.replyToId) }}</div>
									<div v-if="m.text" class="msg-text">{{ m.text }}</div>
									<div v-if="m.image" class="msg-image-wrap"><img :src="m.image" class="msg-image" /></div>
									<div class="react-line">
										<span v-for="r in m.reactions" :key="r.userId" class="react-pill">{{ r.emoji }} {{ r.username }}</span>
									</div>
									<div class="msg-controls">
										<button class="tx-btn mini" @click="replyToId = m.id">Rispondi</button>
										<button class="tx-btn mini" @click="react(m)">Reagisci</button>
										<button class="tx-btn mini" @click="unreact(m)">Rimuovi</button>
										<select class="tx-input mini" @change="forward(m, Number($event.target.value))">
											<option value="">Inoltra…</option>
											<option v-for="c in conversations.filter(x => x.id !== selectedConversationId)" :key="c.id" :value="c.id">
												{{ c.name || ('#' + c.id) }}
											</option>
										</select>
									</div>
								</div>
							</div>
						</section>

						<footer class="composer">
							<div v-if="replyToId" class="replying">Stai rispondendo a #{{ replyToId }}</div>
							<div class="composer-row">
								<input class="tx-input" v-model="text" placeholder="Scrivi un messaggio..." />
							<div class="image-upload-wrap">
								<input class="tx-input" type="file" accept="image/*" @change="onImageFileChange" />
								<small class="muted" v-if="imageFileName">{{ imageFileName }}</small>
								<button class="tx-btn mini" v-if="image" @click="clearImage">Rimuovi immagine</button>
							</div>
								<input class="tx-input short" v-model="reactionEmoji" placeholder="emoji" />
								<button class="tx-btn tx-btn-primary" @click="send">Invia</button>
							</div>
						</footer>
					</div>
					<div v-else class="empty-chat">Seleziona una conversazione per iniziare.</div>
				</main>
			</div>
		</div>
	</div>
</template>

<style scoped>
.wasatx-root {
	min-height: 100vh;
	background: radial-gradient(circle at top, #121b2e 0%, #05070d 40%, #02040a 100%);
	color: #d8e3ff;
	padding: 14px;
}

.main-topbar {
	display: flex;
	justify-content: space-between;
	align-items: center;
	gap: 10px;
	padding: 10px 14px;
	margin-bottom: 12px;
	border-radius: 12px;
	border: 1px solid #1c2d57;
	background: linear-gradient(90deg, #081022 0%, #0b1630 100%);
}

.top-brand {
	font-weight: 800;
	letter-spacing: 0.4px;
}

.top-meta {
	color: #90ade5;
	font-size: 13px;
}

.login-shell {
	min-height: calc(100vh - 28px);
	display: grid;
	place-items: center;
}

.login-panel {
	width: min(560px, 92vw);
	padding: 34px;
	border: 1px solid #1f2e55;
	background: linear-gradient(180deg, #0a0f1e 0%, #070b16 100%);
	border-radius: 18px;
	box-shadow: 0 20px 40px rgba(0, 0, 0, 0.45);
	display: grid;
	gap: 12px;
}

.brand-mark {
	width: 50px;
	height: 50px;
	border-radius: 12px;
	background: #11264f;
	border: 1px solid #27498c;
	display: grid;
	place-items: center;
	font-weight: 800;
}

.chat-shell {
	display: grid;
	grid-template-columns: 300px 340px 1fr;
	gap: 14px;
	min-height: calc(100vh - 94px);
}

.left-rail,
.conversation-rail,
.chat-main {
	border: 1px solid #1c2a4d;
	background: #070d1b;
	border-radius: 14px;
	padding: 12px;
}

.left-rail,
.conversation-rail,
.msg-scroll {
	scrollbar-width: thin;
	scrollbar-color: #2e4f96 #081226;
}

.left-rail::-webkit-scrollbar,
.conversation-rail::-webkit-scrollbar,
.msg-scroll::-webkit-scrollbar {
	width: 10px;
}

.left-rail::-webkit-scrollbar-thumb,
.conversation-rail::-webkit-scrollbar-thumb,
.msg-scroll::-webkit-scrollbar-thumb {
	background: #2e4f96;
	border-radius: 999px;
}

.left-rail { overflow-y: auto; }
.conversation-rail { overflow-y: auto; }
.chat-main { display: flex; }

.panel-card {
	background: #091225;
	border: 1px solid #1a2e58;
	border-radius: 12px;
	padding: 10px;
	margin-top: 10px;
	display: grid;
	gap: 8px;
}

.me-card {
	display: flex;
	gap: 10px;
	align-items: center;
	background: #091225;
	padding: 10px;
	border-radius: 10px;
	border: 1px solid #1a2e58;
	margin-bottom: 6px;
}

.me-name { font-weight: 700; }
.me-user { color: #89a4d9; font-size: 12px; }

.section-title {
	font-size: 12px;
	text-transform: uppercase;
	letter-spacing: 0.8px;
	color: #87a6e5;
}

.avatar {
	width: 42px;
	height: 42px;
	border-radius: 50%;
	display: grid;
	place-items: center;
	background: linear-gradient(145deg, #173573, #0f234a);
	border: 1px solid #385a9c;
	font-weight: 700;
	overflow: hidden;
}

.avatar img {
	width: 100%;
	height: 100%;
	object-fit: cover;
	display: block;
}

.avatar.tiny {
	width: 34px;
	height: 34px;
	font-size: 12px;
}

.user-list, .conv-list { display: grid; gap: 8px; }
.user-row, .conv-row {
	display: grid;
	grid-template-columns: 34px 1fr auto;
	gap: 8px;
	align-items: center;
	background: #0a1224;
	border: 1px solid #142344;
	border-radius: 10px;
	padding: 8px;
}

.conv-row { cursor: pointer; }
.conv-row.active { border-color: #3560ae; background: #0d1931; }
.user-meta,.conv-meta { min-width: 0; }
.user-name,.conv-name { font-weight: 600; }
.user-sub,.conv-preview,.conv-time { color: #7f9bd2; font-size: 12px; }
.conv-preview { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

.rail-head {
	font-weight: 700;
	margin-bottom: 10px;
	padding: 8px;
	background: #0a1429;
	border: 1px solid #203763;
	border-radius: 10px;
}

.chat-wrap {
	display: grid;
	grid-template-rows: auto 1fr auto;
	width: 100%;
	gap: 10px;
}

.chat-head {
	display: flex;
	justify-content: space-between;
	align-items: center;
	gap: 12px;
	background: #0a1224;
	border: 1px solid #15294f;
	border-radius: 10px;
	padding: 10px;
}

.title-main { font-weight: 700; }

.chat-title {
	display: flex;
	align-items: center;
	gap: 8px;
}

.chat-actions {
	display: flex;
	gap: 6px;
	align-items: center;
	flex-wrap: wrap;
	justify-content: flex-end;
}

.msg-scroll {
	overflow-y: auto;
	display: grid;
	gap: 10px;
	padding: 4px;
}

.msg-row {
	display: flex;
	justify-content: flex-start;
}

.msg-row.mine {
	justify-content: flex-end;
}

.msg-bubble {
	max-width: min(72%, 760px);
	background: linear-gradient(180deg, #0b1427 0%, #081021 100%);
	border: 1px solid #203765;
	border-radius: 12px;
	padding: 10px;
	box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.03);
}

.msg-row.mine .msg-bubble {
	background: linear-gradient(180deg, #12284f 0%, #0f2243 100%);
	border-color: #315ea8;
}

.msg-top {
	display: flex;
	justify-content: space-between;
	font-size: 12px;
	margin-bottom: 4px;
}

.msg-text { white-space: pre-wrap; }
.msg-image-wrap { margin-top: 8px; }
.msg-image {
	max-width: 100%;
	max-height: 280px;
	border-radius: 8px;
	border: 1px solid #29457f;
}

.fwd-label,.reply-label {
	font-size: 11px;
	color: #95b1e6;
	margin-bottom: 4px;
}

.react-line { margin-top: 6px; display: flex; flex-wrap: wrap; gap: 4px; }
.react-pill {
	padding: 2px 8px;
	font-size: 12px;
	border-radius: 999px;
	background: #11244a;
	border: 1px solid #274c92;
}

.msg-controls {
	display: flex;
	gap: 6px;
	align-items: center;
	margin-top: 8px;
	flex-wrap: wrap;
}

.composer {
	background: #0a1224;
	border: 1px solid #15294f;
	border-radius: 10px;
	padding: 10px;
}

.composer-row {
	display: grid;
	grid-template-columns: 1fr 1fr 95px auto;
	gap: 8px;
}

.image-upload-wrap {
	display: grid;
	gap: 4px;
}

.replying { color: #9eb8ec; margin-bottom: 8px; font-size: 12px; }
.empty-chat { margin: auto; color: #8fa8db; }
.muted { color: #88a3d8; font-size: 12px; }

.tx-input {
	background: #030812;
	border: 1px solid #223e75;
	color: #e0e9ff;
	border-radius: 10px;
	padding: 8px 10px;
	width: 100%;
}

.tx-input:focus {
	outline: none;
	border-color: #3d69bb;
	box-shadow: 0 0 0 2px rgba(62, 106, 189, 0.2);
}

.tx-input.small { width: auto; min-width: 120px; }
.tx-input.mini { width: auto; min-width: 130px; padding: 5px 8px; }
.tx-input.short { width: 95px; }

.tx-btn {
	background: #142a54;
	border: 1px solid #2f569c;
	color: #e6eeff;
	border-radius: 10px;
	padding: 7px 10px;
	font-weight: 600;
	white-space: nowrap;
}

.tx-btn:hover { filter: brightness(1.1); }
.tx-btn-primary { background: #1d3b77; border-color: #3b69bd; }
.tx-btn-danger { background: #3a1120; border-color: #7d2645; }
.tx-btn.mini { padding: 5px 8px; font-size: 12px; }
.stack-row { display: flex; gap: 8px; }
.login-note { color: #93addf; font-size: 13px; }

@media (max-width: 1500px) {
	.chat-shell { grid-template-columns: 280px 300px 1fr; }
}

@media (max-width: 1250px) {
	.chat-shell { grid-template-columns: 1fr; min-height: auto; }
	.composer-row { grid-template-columns: 1fr; }
	.msg-bubble { max-width: 100%; }
	.main-topbar { flex-direction: column; align-items: flex-start; }
}
</style>
