import { object, ref, string as yupString } from "yup";
import { emailRequired } from "../../utils/validation";
import { useCustomForm } from "../../hooks/useCustomForm";
import { InputField } from "../../components/fields/InputField";
import { Button } from "../../components/buttons/Button";
import { useMutation } from "@tanstack/react-query";
import { authService } from "../../services/auth";
import toast from "react-hot-toast";
import { getErrorMessage } from "../../utils/errors";
import type { FC } from "react";

const schema = object({
  email: emailRequired("Email is required"),
  password: yupString()
    .required("Password is required")
    .min(8, "Password must be at least 8 characters long"),

  confirmPassword: yupString()
    .required("Please confirm your password")
    .oneOf([ref("password")], "Passwords do not match"),
});

export const RegisterPage: FC<{ refresh: () => void }> = ({ refresh }) => {
  const form = useCustomForm(schema);

  const mutation = useMutation({
    mutationFn: authService.setup,
    onSuccess: () => {
      toast.success("Setup completed successfully");
      refresh();
    },
    onError: error => toast.error(getErrorMessage(error)),
  });

  const onSubmit = form.handleSubmit(formData => mutation.mutate(formData));

  return (
    <div className="flex items-center justify-center min-h-screen px-4 bg-base-200">
      <div className="card w-full max-w-md shadow-xl bg-base-100">
        <div className="card-body">
          <h1 className="text-3xl font-bold text-center">Create account</h1>
          <form onSubmit={onSubmit} className="space-y-4">
            <InputField
              label="Email"
              registration={form.register("email")}
              autoComplete="off"
              type="email"
              error={form.formState.errors.email}
            />
            <InputField
              label="Password"
              type="password"
              registration={form.register("password")}
              autoComplete="off"
              error={form.formState.errors.password}
            />
            <InputField
              label="Confirm password"
              type="password"
              registration={form.register("confirmPassword")}
              autoComplete="off"
              error={form.formState.errors.confirmPassword}
            />
            <div className="form-control mt-6">
              <Button
                type="submit"
                label="Register"
                loading={mutation.isPending}
              />
            </div>
          </form>
        </div>
      </div>
    </div>
  );
};
