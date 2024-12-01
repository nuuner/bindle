export const config = {
    apiHost:
        import.meta.env.MODE === "development"
            ? "http://localhost:3000/api"
            : "/api",
    contactEmail: import.meta.env.VITE_CONTACT_EMAIL,
};
