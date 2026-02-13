package api

import "net/http"


func (rt *_router) Handler() http.Handler {
	rt.router.GET("/liveness", rt.liveness)

	rt.router.POST("/api/login", rt.wrap(rt.login))
	rt.router.POST("/api/logout", rt.wrap(rt.logout))
	rt.router.GET("/api/me", rt.wrap(rt.getMe))
	rt.router.PUT("/api/me", rt.wrap(rt.updateMe))
	rt.router.PUT("/api/me/photo", rt.wrap(rt.updateMyPhoto))
	rt.router.GET("/api/users", rt.wrap(rt.listUsers))

	rt.router.GET("/api/conversations", rt.wrap(rt.listConversations))
	rt.router.POST("/api/conversations/direct", rt.wrap(rt.createDirectConversation))
	rt.router.POST("/api/conversations/group", rt.wrap(rt.createGroupConversation))
	rt.router.GET("/api/conversation/:id", rt.wrap(rt.getConversation))
	rt.router.PUT("/api/conversation/:id", rt.wrap(rt.updateConversation))
	rt.router.PUT("/api/conversation/:id/photo", rt.wrap(rt.updateGroupPhoto))
	rt.router.POST("/api/conversation/:id/add", rt.wrap(rt.addToGroup))
	rt.router.POST("/api/conversation/:id/leave", rt.wrap(rt.leaveGroup))
	rt.router.POST("/api/conversation/:id/messages", rt.wrap(rt.sendMessage))

	rt.router.POST("/api/messages/:id/read", rt.wrap(rt.markRead))
	rt.router.POST("/api/messages/:id/reaction", rt.wrap(rt.setReaction))
	rt.router.DELETE("/api/messages/:id/reaction", rt.wrap(rt.removeReaction))
	rt.router.POST("/api/messages/:id/forward", rt.wrap(rt.forwardMessage))
	rt.router.DELETE("/api/messages/:id", rt.wrap(rt.deleteMessage))

	return rt.router
}
