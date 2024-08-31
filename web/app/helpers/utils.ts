export const fetcher = (resource: string) => fetch(`${process.env.NEXT_PUBLIC_API_URL}${resource}`)
    .then((res) => res.json());
