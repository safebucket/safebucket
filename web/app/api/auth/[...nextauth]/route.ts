import NextAuth, { Account, Profile } from "next-auth";
import AppleProvider from "next-auth/providers/apple";
import GoogleProvider from "next-auth/providers/google";

const handler = NextAuth({
  pages: {
    signIn: "/login",
    signOut: "/auth/signout",
    error: "/auth/error", // Error code passed in query string as ?error=
    verifyRequest: "/auth/verify-request", // (used for check email message)
    newUser: "/auth/new-user", // New users will be directed here on first sign in (leave the property out if not of interest)
  },
  providers: [
    GoogleProvider({
      clientId: process.env.GOOGLE_CLIENT_ID as string,
      clientSecret: process.env.GOOGLE_CLIENT_SECRET as string,
    }),
    AppleProvider({
      clientId: "CHANGEME",
      clientSecret: "CHANGEME",
    }),
  ],
  callbacks: {
    async redirect({ baseUrl }) {
      return baseUrl;
    },
    async signIn({
      account,
      profile,
    }: {
      account: Account | null;
      profile?: Profile | undefined;
    }) {
      const isAccount = account && profile;
      console.log(account);
      console.log(profile);
      if (isAccount && account.provider === "google") {
        // return profile.email_verified && profile.email.endsWith("@example.com");
        return true;
      }
      return true; // Handle other providers differently
    },
  },
});

export { handler as GET, handler as POST };
