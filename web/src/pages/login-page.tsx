import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

export function LoginPage() {
  return (
    <div className="mx-auto flex min-h-screen max-w-md items-center px-4 py-8">
      <Card className="w-full">
        <CardHeader>
          <CardTitle>Login</CardTitle>
          <CardDescription>
            Sign in to access the Get Rich Quick frontend. Authenticated users are redirected to the dashboard automatically.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            The login form will be added in the full authentication flow. This route already protects the app shell from
            unauthenticated access.
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
