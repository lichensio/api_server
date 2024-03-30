package http

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func NewRouter(svc *Service) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.StripSlashes)

	r.Route("/api", func(r chi.Router) {
		r.Post("/loadEmployees", svc.LoadEmployeesHandler)
		r.Get("/db/create", svc.DBCreateHandler)
		r.Delete("/db/delete", svc.DBDeleteHandler)
		r.Get("/getMonthlySchedule", svc.GetMonthlyScheduleHandler)
		r.Get("/getEmployees", svc.GetEmployeesHandler)
		r.Get("/getWeeksAB/{ID}", svc.GetWeeksABHandler)
		// r.Put("/updateEmployees", svc.UpdateEmployees)
		// r.Put("/updateSchedule", svc.UpdateSchedule)
		// r.Get("/getSchedule/{employeeID}", svc.GetSchedule)
		// r.Get("/getEmployees", svc.GetEmployees)
		// r.Get("/getCalendar/{year}/{month}", svc.GetCalendar)
		// r.Get("/analytics", svc.GetAnalytics)
	})

	return r
}
